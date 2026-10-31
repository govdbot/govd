package facebook

import (
	"errors"
	"fmt"
	"github.com/govdbot/govd/internal/util"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
)

var facebookHost = []string{"facebook"}

var shareURLPattern = regexp.MustCompile(`https?://(?:(?:www|m)\.)?facebook\.com/share/(?:r|v|p)/(?P<id>[a-zA-Z0-9]+)`)

var ShareExtractor = &models.Extractor{
	ID:          "facebook",
	DisplayName: "Facebook (Share)",

	URLPattern: shareURLPattern,
	Host:       facebookHost,

	Redirect: false,
	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := GetPublicMedia(ctx)
		return &models.ExtractorResponse{Media: media}, err
	},
}

var videoURLPattern = regexp.MustCompile(
	`https?://(?:(?:www|m|mbasic)\.)?facebook\.com/` +
		`(?:watch/?\?(?:[^&]*&)*v=|video(?:/video)?\.php\?(?:[^&]*&)*v=|(?:reel|videos?|posts?)/|[^/]+/(?:videos|posts|reels?)/)` +
		`(?P<id>[a-zA-Z0-9]+)`,
)

var Extractor = &models.Extractor{
	ID:          "facebook",
	DisplayName: "Facebook",

	URLPattern: videoURLPattern,
	Host:       facebookHost,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := GetMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			Media: media,
		}, nil
	},
}

func GetMedia(ctx *models.ExtractorContext) (*models.Media, error) {

	media, publicErr := GetPublicMedia(ctx)
	if publicErr == nil {
		return media, nil
	}
	if errors.Is(publicErr, util.ErrFileTooLarge) || errors.Is(publicErr, util.ErrDurationTooLong) || len(ctx.HTTPClient.Cookies) == 0 {
		return nil, publicErr
	}
	videoData, err := GetVideoData(ctx)
	if err != nil {
		return nil, fmt.Errorf("Facebook public/native methods failed: %w; %w", publicErr, err)
	}
	return buildMedia(ctx, videoData)
}

func buildMedia(ctx *models.ExtractorContext, data *VideoData) (*models.Media, error) {
	media := ctx.NewMedia()
	if data.Title != "" {
		media.SetCaption(data.Title)
	}

	item := media.NewItem()
	var formats []*models.MediaFormat

	if data.HDURL != "" {
		formats = append(formats, &models.MediaFormat{
			FormatID:   "hd",
			Type:       database.MediaTypeVideo,
			VideoCodec: database.MediaCodecAvc,
			AudioCodec: database.MediaCodecAac,
			URL:        []string{data.HDURL},
			Width:      data.Width,
			Height:     data.Height,
		})
	}
	if data.SDURL != "" {
		formats = append(formats, &models.MediaFormat{
			FormatID:   "sd",
			Type:       database.MediaTypeVideo,
			VideoCodec: database.MediaCodecAvc,
			AudioCodec: database.MediaCodecAac,
			URL:        []string{data.SDURL},
		})
	}

	if len(formats) == 0 {
		return nil, fmt.Errorf("no video formats found")
	}

	item.AddFormats(formats...)
	return media, nil
}
