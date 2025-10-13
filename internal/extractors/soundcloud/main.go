package soundcloud

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/plugins"
	"github.com/govdbot/govd/internal/util"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

const (
	apiHostname = "https://api-v2.soundcloud.com/"
	baseURL     = "https://soundcloud.com/"
)

var ShortExtractor = &models.Extractor{
	ID:          "soundcloud",
	DisplayName: "SoundCloud (Short)",

	URLPattern: regexp.MustCompile(`https?:\/\/on\.soundcloud\.com\/(?P<id>\w+)`),
	Host:       []string{"soundcloud"},

	Redirect: true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		redirectURL, err := ctx.FetchLocation(ctx.ContentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get url location: %w", err)
		}
		return &models.ExtractorResponse{
			URL: redirectURL,
		}, nil
	},
}

var Extractor = &models.Extractor{
	ID:          "soundcloud",
	DisplayName: "SoundCloud",

	URLPattern: regexp.MustCompile(`(?i)^(?:https?://)?(?:(?:www\.|m\.)?soundcloud\.com/(?P<uploader>[\w\d-]+)/(?P<id>[\w\d-]+)(?:/(?P<token>[^/?#]+))?(?:[?].*)?$|api(?:-v2)?\.soundcloud\.com/tracks/(?P<track_id>\d+)(?:/?\?secret_token=(?P<secret_token>[^&]+))?)`),
	Host:       []string{"soundcloud"},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := GetTrackMedia(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get media: %w", err)
		}
		return &models.ExtractorResponse{
			Media: media,
		}, nil
	},
}

func GetTrackMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	var infoURL string
	var query = make(map[string]string)

	contentID := ctx.ContentID
	trackID := ctx.MatchGroups["track_id"]

	if trackID != "" {
		infoURL = apiHostname + "tracks/" + trackID
		contentID = trackID
		token := ctx.MatchGroups["secret_token"]
		if token != "" {
			query["secret_token"] = token
		}
	} else {
		uploader := ctx.MatchGroups["uploader"]
		resolveTitle := uploader + "/" + contentID
		token := ctx.MatchGroups["token"]
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
		return nil, fmt.Errorf("no suitable format found")
	}

	trackManifest, err := GetTrackURL(ctx, formatObj.URL, clientID)
	if err != nil {
		return nil, err
	}

	media := ctx.NewMedia()
	item := media.NewItem()
	item.AddFormats(&models.MediaFormat{
		FormatID:     "mp3",
		Type:         database.MediaTypeAudio,
		AudioCodec:   database.MediaCodecMp3,
		URL:          []string{trackManifest.URL},
		ThumbnailURL: []string{thumbnail},
		Duration:     duration,
		Title:        title,
		Artist:       artist,
		Plugins:      []*models.Plugin{plugins.ID3},
	})

	return media, nil
}

func GetTrackManifest(
	ctx *models.ExtractorContext,
	trackURL string,
	query map[string]string,
	clientID string,
) (*Track, error) {
	queryParams := url.Values{}
	for k, v := range query {
		queryParams[k] = []string{v}
	}
	queryParams["client_id"] = []string{clientID}
	reqURL := trackURL + "&" + queryParams.Encode()

	zap.S().Debugf("manifest URL: %s", reqURL)

	resp, err := ctx.Fetch(
		http.MethodGet,
		reqURL, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

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
	ctx *models.ExtractorContext,
	trackURL string,
	clientID string,
) (*TrackManifest, error) {
	reqURL := trackURL + "?client_id=" + clientID

	resp, err := ctx.Fetch(
		http.MethodGet,
		reqURL, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

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
