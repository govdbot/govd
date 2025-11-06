package tiktok

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

var VMExtractor = &models.Extractor{
	ID:          "tiktok",
	DisplayName: "TikTok VM",

	URLPattern: regexp.MustCompile(`https:\/\/((?:vm|vt|www)\.)?(vx)?tiktok\.com\/(?:t\/)?(?P<id>[a-zA-Z0-9-]+)`),
	Host:       []string{"tiktok", "vxtiktok"},
	Redirect:   true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		redirectURL, err := ctx.FetchLocation(ctx.ContentURL, nil)
		if err != nil {
			return nil, err
		}
		parsedURL, err := url.Parse(redirectURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redirect url: %w", err)
		}

		if parsedURL.Path == "/login" {
			logger.L.Debug("tiktok is geo restricted in your region, attemping bypass...")
			realURL := parsedURL.Query().Get("redirect_url")
			if realURL == "" {
				return nil, util.ErrGeoRestrictedContent
			}
			logger.L.Debugf("found url: %s", realURL)
			return &models.ExtractorResponse{
				URL: realURL,
			}, nil
		}
		return &models.ExtractorResponse{
			URL: redirectURL,
		}, nil
	},
}

var Extractor = &models.Extractor{
	ID:          "tiktok",
	DisplayName: "TikTok",

	URLPattern: regexp.MustCompile(`https?:\/\/((www|m)\.)?(vx)?tiktok\.com\/((?:embed|@[\w\.-]*)\/)?(v(ideo)?|p(hoto)?)\/(?P<id>[0-9]+)`),
	Host:       []string{"tiktok", "vxtiktok"},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := GetMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			URL:   ctx.ContentURL,
			Media: media,
		}, nil
	},
}

func GetMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	var details *WebItemStruct
	var cookies []*http.Cookie
	var err error

	// sometimes web page just returns a
	// login page, so we need to retry
	// a few times to get the correct page
	for range 5 {
		details, cookies, err = GetVideoWeb(ctx)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get from web: %w", err)
	}

	media := ctx.NewMedia()
	media.SetCaption(details.Desc)

	isImageSlide := details.ImagePost != nil
	if !isImageSlide {
		item := media.NewItem()
		video := details.Video
		if video.PlayAddr != nil {
			item.AddFormats(&models.MediaFormat{
				Type:       database.MediaTypeVideo,
				FormatID:   video.PlayAddr.URI,
				URL:        video.PlayAddr.URLList,
				VideoCodec: database.MediaCodecAvc,
				AudioCodec: database.MediaCodecAac,
				Width:      video.PlayAddr.Width,
				Height:     video.PlayAddr.Height,
				Duration:   video.Duration,
				DownloadSettings: &models.DownloadSettings{
					// avoid 403 error for videos
					Cookies: cookies,
				},
			})
			return media, nil
		}
		return nil, fmt.Errorf("no video formats found")
	} else {
		images := details.ImagePost.Images
		for _, image := range images {
			item := media.NewItem()
			item.AddFormats(&models.MediaFormat{
				Type:     database.MediaTypePhoto,
				FormatID: "image",
				URL:      image.URL.URLList,
			})
		}
		return media, nil
	}
}
