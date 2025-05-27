package instagram

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/logger"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/networking"

	"go.uber.org/zap"
)

var instagramHost = []string{
	"instagram",
	"ddinstagram",
}

var Extractor = &models.Extractor{
	Name:       "Instagram",
	CodeName:   "instagram",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/(www\.)?(?:dd)?instagram\.com\/(reels?|p|tv)\/(?P<id>[a-zA-Z0-9_-]+)`),
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
		zap.S().Debugf(
			"failed to get media from GQL API: %v",
			err,
		)
		// method 2: get media from embed page
		mediaList, err = GetEmbedMediaList(ctx)
		if err == nil && len(mediaList) > 0 {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		zap.S().Debugf(
			"failed to get media from embed page: %v",
			err,
		)
		// method 3: get media from 3rd party service (unlikely)
		mediaList, err = GetIGramMediaList(ctx)
		if err == nil && len(mediaList) > 0 {
			return &models.ExtractorResponse{
				MediaList: mediaList,
			}, nil
		}
		zap.S().Debugf(
			"failed to get media from 3rd party service: %v",
			err,
		)
		return nil, ErrAllMethodsFailed
	},
}

var StoriesExtractor = &models.Extractor{
	Name:       "Instagram Stories",
	CodeName:   "instagram",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/(www\.)?(?:dd)?instagram\.com\/stories\/[a-zA-Z0-9._]+\/(?P<id>\d+)`),
	Host:       instagramHost,
	IsRedirect: false,
	IsHidden:   true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		mediaList, err := GetIGramMediaList(ctx)
		return &models.ExtractorResponse{
			MediaList: mediaList,
		}, err
	},
}

var ShareURLExtractor = &models.Extractor{
	Name:       "Instagram Share URL",
	CodeName:   "instagram",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?:\/\/(www\.)?(?:dd)?instagram\.com\/share\/((reels?|video|s|p)\/)?(?P<id>[^\/\?]+)`),
	Host:       instagramHost,
	IsRedirect: true,
	IsHidden:   true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		client := networking.GetExtractorHTTPClient(ctx.Extractor)
		cookies := util.GetExtractorCookies(ctx.Extractor)
		redirectURL, err := util.GetLocationURL(
			client,
			ctx.MatchedContentURL,
			webHeaders,
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
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	embedURL := fmt.Sprintf(
		"https://www.instagram.com/p/%s/embed/captioned",
		ctx.MatchedContentID,
	)
	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		embedURL,
		nil,
		webHeaders,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("ig_embed_response", resp)

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
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	apiURL := fmt.Sprintf(
		"https://%s/api/convert",
		igramHostname,
	)
	payload, err := BuildIGramPayload(contentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build signed payload: %w", err)
	}
	resp, err := util.FetchPage(
		client,
		http.MethodPost,
		apiURL,
		payload,
		igramHeaders,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("ig_3party_response", resp)

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
