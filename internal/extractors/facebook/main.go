package facebook

import (
	"fmt"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

var facebookHost = []string{"facebook"}

var ShareExtractor = &models.Extractor{
	ID:          "facebook",
	DisplayName: "Facebook (Share)",

	URLPattern: regexp.MustCompile(`https?://(?:(?:www|m)\.)?facebook\.com/share/(?:r|v|p)/(?P<id>[a-zA-Z0-9]+)`),
	Host:       facebookHost,

	Redirect: true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		finalURL, err := ctx.FetchLocation(
			ctx.ContentURL,
			&networking.RequestParams{Headers: webHeaders},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to follow share redirect: %w", err)
		}
		return &models.ExtractorResponse{URL: finalURL}, nil
	},
}

var Extractor = &models.Extractor{
	ID:          "facebook",
	DisplayName: "Facebook",

	URLPattern: regexp.MustCompile(
		`https?://(?:(?:www|m|mbasic)\.)?facebook\.com/` +
			`(?:watch/?\?(?:[^&]*&)*v=|(?:reel|videos?|posts?)/|[^/]+/(?:videos|posts|reels?)/)` +
			`(?P<id>[a-zA-Z0-9]+)`,
	),
	Host: facebookHost,

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
	if ctx.HTTPClient.Cookies == nil {
		return nil, fmt.Errorf("auth cookies are required for facebook")
	}
	videoData, err := GetVideoData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get video data: %w", err)
	}
	return buildMedia(ctx, videoData)
}

// Bitrate hints used to keep the format selector deterministic. The
// Facebook page does not expose real bitrates, but we still need to
// give the sort comparator (highest bitrate wins) a stable signal so
// HD ranks above SD when both fit.
const (
	hdBitrateHint int64 = 4_500_000
	sdBitrateHint int64 = 1_500_000
)

func buildMedia(ctx *models.ExtractorContext, data *VideoData) (*models.Media, error) {
	media := ctx.NewMedia()
	if data.Title != "" {
		media.SetCaption(data.Title)
	}

	item := media.NewItem()

	hdSize := probeFormatSize(ctx, data.HDURL)
	sdSize := probeFormatSize(ctx, data.SDURL)
	tgLimit := util.TelegramBotAPIFileLimit()

	hdFits := data.HDURL != "" && (hdSize == 0 || hdSize <= tgLimit)
	sdFits := data.SDURL != "" && (sdSize == 0 || sdSize <= tgLimit)

	ctx.Infof(
		"facebook formats: hd=%dB(fits=%t) sd=%dB(fits=%t) tg_limit=%dB",
		hdSize, hdFits, sdSize, sdFits, tgLimit,
	)

	var formats []*models.MediaFormat

	if hdFits {
		formats = append(formats, &models.MediaFormat{
			FormatID:   "hd",
			Type:       database.MediaTypeVideo,
			VideoCodec: database.MediaCodecAvc,
			AudioCodec: database.MediaCodecAac,
			URL:        []string{data.HDURL},
			Width:      data.Width,
			Height:     data.Height,
			Bitrate:    hdBitrateHint,
			FileSize:   hdSize,
		})
	}
	if sdFits {
		formats = append(formats, &models.MediaFormat{
			FormatID:   "sd",
			Type:       database.MediaTypeVideo,
			VideoCodec: database.MediaCodecAvc,
			AudioCodec: database.MediaCodecAac,
			URL:        []string{data.SDURL},
			Bitrate:    sdBitrateHint,
			FileSize:   sdSize,
		})
	}

	if len(formats) == 0 {
		// Both formats either failed to extract or exceed the active
		// Bot API upload limit. Surface the friendlier "too large"
		// error if we know at least one format was just oversized so
		// the user understands why.
		if (data.HDURL != "" && hdSize > tgLimit) || (data.SDURL != "" && sdSize > tgLimit) {
			return nil, util.ErrTelegramFileTooLarge
		}
		return nil, fmt.Errorf("no video formats found")
	}

	item.AddFormats(formats...)
	return media, nil
}
