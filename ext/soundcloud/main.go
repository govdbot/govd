package soundcloud

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/logger"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/plugins"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/networking"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

const (
	apiHostname = "https://api-v2.soundcloud.com/"
	baseURL     = "https://soundcloud.com/"
)

var ShortExtractor = &models.Extractor{
	Name:       "SoundCloud (Short)",
	CodeName:   "soundcloud",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategoryMusic,
	URLPattern: regexp.MustCompile(`https?:\/\/on\.soundcloud\.com\/(?P<id>\w+)`),
	Host:       []string{"on.soundcloud"},
	IsRedirect: true,
	IsHidden:   true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		client := networking.GetExtractorHTTPClient(ctx.Extractor)
		cookies := util.GetExtractorCookies(ctx.Extractor)

		redirectURL, err := util.GetLocationURL(
			client,
			ctx.MatchedContentURL,
			nil,
			cookies,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get url location: %w", err)
		}

		return &models.ExtractorResponse{
			URL: redirectURL,
		}, nil
	},
}

var Extractor = &models.Extractor{
	Name:       "SoundCloud",
	CodeName:   "soundcloud",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategoryMusic,
	URLPattern: regexp.MustCompile(`(?i)^(?:https?://)?(?:(?:www\.|m\.)?soundcloud\.com/(?P<uploader>[\w\d-]+)/(?P<id>[\w\d-]+)(?:/(?P<token>[^/?#]+))?(?:[?].*)?$|api(?:-v2)?\.soundcloud\.com/tracks/(?P<track_id>\d+)(?:/?\?secret_token=(?P<secret_token>[^&]+))?)`),
	Host:       []string{"soundcloud"},

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		mediaList, err := GetTrackMediaList(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get media: %w", err)
		}
		return &models.ExtractorResponse{
			MediaList: mediaList,
		}, nil
	},
}

func GetTrackMediaList(ctx *models.DownloadContext) ([]*models.Media, error) {
	var infoURL string
	var query = make(map[string]string)

	contentID := ctx.MatchedContentID
	trackID := ctx.MatchedGroups["track_id"]

	if trackID != "" {
		infoURL = apiHostname + "tracks/" + trackID
		contentID = trackID
		token := ctx.MatchedGroups["secret_token"]
		if token != "" {
			query["secret_token"] = token
		}
	} else {
		uploader := ctx.MatchedGroups["uploader"]
		resolveTitle := uploader + "/" + contentID
		token := ctx.MatchedGroups["token"]
		if token != "" {
			resolveTitle += "/" + token
		}
		infoURL = ResolveURL(baseURL + resolveTitle)
	}

	clientID, err := GetClientID(ctx)
	if err != nil {
		return nil, err
	}

	manifest, err := GetTrackManifest(ctx, infoURL, query, clientID)
	if err != nil {
		return nil, err
	}

	title := manifest.Title
	artist := manifest.User.Username
	thumbnail := GetThumbnailURL(manifest.ArtworkURL)
	duration := manifest.FullDuration / 1000

	var formatObj *Transcoding
	for _, fmt := range manifest.Media.Transcodings {
		if regexp.MustCompile(`^mp3`).MatchString(fmt.Preset) && fmt.Format.Protocol == "progressive" {
			formatObj = fmt
			break
		}
	}

	if formatObj == nil {
		return nil, ErrNoSuitableFormat
	}

	trackManifest, err := GetTrackURL(ctx, formatObj.URL, clientID)
	if err != nil {
		return nil, err
	}

	media := ctx.Extractor.NewMedia(contentID, ctx.MatchedContentURL)
	media.SetCaption(title)

	media.AddFormat(&models.MediaFormat{
		FormatID:   "mp3",
		Type:       enums.MediaTypeAudio,
		AudioCodec: enums.MediaCodecMP3,
		URL:        []string{trackManifest.URL},
		Duration:   duration,
		Thumbnail:  []string{thumbnail},
		Title:      title,
		Artist:     artist,
		Plugins: []models.Plugin{
			plugins.SetID3,
		},
		DownloadConfig: &models.DownloadConfig{
			Remux: false,
		},
	})

	return []*models.Media{media}, nil
}

func GetTrackManifest(
	ctx *models.DownloadContext,
	trackURL string,
	query map[string]string,
	clientID string,
) (*Track, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	queryParams := url.Values{}
	for k, v := range query {
		queryParams[k] = []string{v}
	}
	queryParams["client_id"] = []string{clientID}
	reqURL := trackURL + "&" + queryParams.Encode()

	zap.S().Debugf("manifest URL: %s", reqURL)

	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		reqURL,
		nil,
		nil,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("soundcloud_manifest_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get track info: %s", resp.Status)
	}

	var track Track
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&track)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &track, nil
}

func GetTrackURL(
	ctx *models.DownloadContext,
	trackURL string,
	clientID string,
) (*TrackManifest, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	reqURL := trackURL + "?client_id=" + clientID

	zap.S().Debugf("soundcloud track url: %s", reqURL)

	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		reqURL,
		nil,
		nil,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("soundcloud_track_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get track URL: %s", resp.Status)
	}

	var manifest TrackManifest
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if strings.Contains(manifest.URL, "preview-media") {
		return nil, util.ErrPaidContent
	}

	return &manifest, nil
}
