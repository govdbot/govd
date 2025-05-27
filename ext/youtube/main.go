package youtube

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/logger"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/networking"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

var Extractor = &models.Extractor{
	Name:       "YouTube",
	CodeName:   "youtube",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategoryStreaming,
	URLPattern: regexp.MustCompile(`(?:https?:)?(?:\/\/)?(?:(?:www|m)\.)?(?:youtube(?:-nocookie)?\.com\/(?:(?:watch\?(?:.*&)?v=)|(?:embed\/)|(?:v\/)|(?:shorts\/))|youtu\.be\/)(?P<id>[\w-]{11})(?:[?&].*)?`),
	Host: []string{
		"youtube",
		"youtu",
		"youtube-nocookie",
	},

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		video, err := GetVideoFromInv(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			MediaList: []*models.Media{video},
		}, nil
	},
}

func GetVideoFromInv(ctx *models.DownloadContext) (*models.Media, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	cfg := config.GetExtractorConfig(ctx.Extractor)
	if cfg == nil {
		return nil, ErrNotConfigured
	}
	instance, err := GetInvInstance(cfg)
	if err != nil {
		return nil, err
	}

	videoID := ctx.MatchedContentID
	videoURL := ctx.MatchedContentURL
	media := ctx.Extractor.NewMedia(
		videoID,
		videoURL,
	)

	reqURL := instance +
		invEndpoint +
		videoID +
		"?local=true" // proxied CDN

	zap.S().Debugf("proxied invidious api: %s", reqURL)

	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		reqURL,
		nil,
		nil,
		cookies,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("inv_youtube_response", resp)

	var data *InvResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	switch data.Error {
	case "This video may be inappropriate for some users.":
		return nil, ErrAgeRestricted
	default:
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad response: %s", resp.Status)
		}
	}

	formats := ParseInvFormats(data)
	if len(formats) == 0 {
		return nil, ErrNoValidFormats
	}
	media.SetCaption(data.Title)
	for _, format := range formats {
		media.AddFormat(format)
	}
	return media, nil
}
