package instagram

import (
	"fmt"
	"govd/enums"
	"govd/models"
	"govd/util"
	"io"
	"net/http"
	"regexp"

	"github.com/pkg/errors"
)

var instagramHost = []string{"instagram.com"}

var Extractor = &models.Extractor{
	Name:       "Instagram",
	CodeName:   "instagram",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/(www\.)?instagram\.com\/(reels?|p|tv)\/(?P<id>[a-zA-Z0-9_-]+)`),
	Host:       instagramHost,
	IsRedirect: false,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		// method 1: get media from GQL web API
		mediaList, err := GetGQLMediaList(ctx)
		if err == nil && len(mediaList) > 0 {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		// method 2: get media from embed page
		mediaList, err = GetEmbedMediaList(ctx)
		if err == nil && len(mediaList) > 0 {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		// method 3: get media from 3rd party service (unlikely)
		mediaList, err = GetIGramMediaList(ctx)
		if err == nil && len(mediaList) > 0 {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		return nil, errors.New("failed to extract media: all methods failed")
	},
}

var StoriesExtractor = &models.Extractor{
	Name:       "Instagram Stories",
	CodeName:   "instagram_stories",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/(www\.)?instagram\.com\/stories\/[a-zA-Z0-9._]+\/(?P<id>\d+)`),
	Host:       instagramHost,
	IsRedirect: false,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		mediaList, err := GetIGramMediaList(ctx)
		return &models.ExtractorResponse{
			MediaList: mediaList,
		}, err
	},
}

var ShareURLExtractor = &models.Extractor{
	Name:       "Instagram Share URL",
	CodeName:   "instagram_share",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?:\/\/(www\.)?instagram\.com\/share\/((reels?|video|s|p)\/)?(?P<id>[^\/\?]+)`),
	Host:       instagramHost,
	IsRedirect: true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		client := util.GetHTTPClient(ctx.Extractor.CodeName)
		redirectURL, err := util.GetLocationURL(
			client,
			ctx.MatchedContentURL,
			igHeaders,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get url location: %w", err)
		}
		return &models.ExtractorResponse{
			URL: redirectURL,
		}, nil
	},
}

func GetGQLMediaList(
	ctx *models.DownloadContext,
) ([]*models.Media, error) {
	graphData, err := GetGQLData(ctx, ctx.MatchedContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get graph data: %w", err)
	}
	return ParseGQLMedia(ctx, graphData.ShortcodeMedia)
}

func GetEmbedMediaList(
	ctx *models.DownloadContext,
) ([]*models.Media, error) {
	session := util.GetHTTPClient(ctx.Extractor.CodeName)
	embedURL := fmt.Sprintf("https://www.instagram.com/p/%s/embed/captioned", ctx.MatchedContentID)
	req, err := http.NewRequest(
		http.MethodGet,
		embedURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range igHeaders {
		req.Header.Set(key, value)
	}
	resp, err := session.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get embed page: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	graphData, err := ParseEmbedGQL(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse embed page: %w", err)
	}
	return ParseGQLMedia(ctx, graphData)
}

func GetIGramMediaList(ctx *models.DownloadContext) ([]*models.Media, error) {
	postURL := ctx.MatchedContentURL
	details, err := GetFromIGram(ctx, postURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	mediaList := make([]*models.Media, 0, len(details.Items))
	for _, item := range details.Items {
		media := ctx.Extractor.NewMedia(
			ctx.MatchedContentID,
			ctx.MatchedContentURL,
		)
		urlObj := item.URL[0]
		contentURL, err := GetCDNURL(urlObj.URL)
		if err != nil {
			return nil, err
		}
		thumbnailURL, err := GetCDNURL(item.Thumb)
		if err != nil {
			return nil, err
		}
		fileExt := urlObj.Ext
		formatID := urlObj.Type
		switch fileExt {
		case "mp4":
			media.AddFormat(&models.MediaFormat{
				Type:       enums.MediaTypeVideo,
				FormatID:   formatID,
				URL:        []string{contentURL},
				VideoCodec: enums.MediaCodecAVC,
				AudioCodec: enums.MediaCodecAAC,
				Thumbnail:  []string{thumbnailURL},
			},
			)
		case "jpg", "webp", "heic", "jpeg":
			media.AddFormat(&models.MediaFormat{
				Type:     enums.MediaTypePhoto,
				FormatID: formatID,
				URL:      []string{contentURL},
			})
		default:
			return nil, fmt.Errorf("unknown format: %s", fileExt)
		}
		mediaList = append(mediaList, media)
	}

	return mediaList, nil
}

func GetFromIGram(
	ctx *models.DownloadContext,
	contentURL string,
) (*IGramResponse, error) {
	session := util.GetHTTPClient(ctx.Extractor.CodeName)
	apiURL := fmt.Sprintf(
		"https://%s/api/convert",
		igramHostname,
	)
	payload, err := BuildIGramPayload(contentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build signed payload: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, apiURL, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", util.ChromeUA)

	resp, err := session.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	response, err := ParseIGramResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return response, nil
}
