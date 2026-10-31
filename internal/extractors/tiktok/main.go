package tiktok

import (
	"fmt"
	"github.com/govdbot/govd/internal/networking"
	"net/http"
	"net/url"
	"regexp"

	"github.com/govdbot/govd/internal/database"
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
		redirectURL, err := ctx.FetchLocation(ctx.ContentURL, &networking.RequestParams{Headers: publicTikTokHeaders(), StopRedirect: func(req *http.Request) bool {
			if !publicTikTokURL(req.URL.String()) {
				return true
			}
			if idx := Extractor.URLPattern.FindStringIndex(req.URL.String()); idx != nil && idx[0] == 0 {
				return true
			}
			return false
		}})
		if err != nil {
			return nil, err
		}
		parsedURL, err := url.Parse(redirectURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redirect url: %w", err)
		}
		if !publicTikTokURL(redirectURL) {
			return nil, networking.ErrUnsafeDestination
		}

		if parsedURL.Path == "/login" {
			ctx.Debugf("tiktok is geo restricted in your region, attemping bypass...")
			realURL := parsedURL.Query().Get("redirect_url")
			if !publicTikTokURL(realURL) || !Extractor.URLPattern.MatchString(realURL) {
				return nil, util.ErrGeoRestrictedContent
			}
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

	URLPattern: regexp.MustCompile(`https?:\/\/(?:(?:www|m)\.)?(?:vx)?tiktok\.com\/(?:(?:@[\w.-]*\/)?(?:video|photo|v|p)\/|embed\/(?:v2\/)?)(?P<id>[0-9]+)`),
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
	details, cookies, webErr := GetVideoWeb(ctx)
	if webErr == nil {
		webErr = ensureItemIdentity(details, ctx.ContentID)
	}
	if webErr == nil && details.ImagePost == nil && !hasDownloadableVideo(details.Video) {
		webErr = util.ErrUnavailable
	}
	if webErr != nil {
		var embedErr error
		details, embedErr = GetVideoEmbed(ctx)
		cookies = nil
		if embedErr != nil {
			return nil, fmt.Errorf("TikTok public web/embed failed: %w; %w", webErr, embedErr)
		}
	}
	if err := ensureItemIdentity(details, ctx.ContentID); err != nil {
		return nil, err
	}

	media := ctx.NewMedia()
	media.SetCaption(details.Desc)

	isImageSlide := details.ImagePost != nil
	if !isImageSlide {
		item := media.NewItem()
		video := details.Video
		if video == nil {
			return nil, util.ErrUnavailable
		}
		if video.PlayAddr == nil {
			urls := validTikTokURLs([]string{video.PlayURL, video.DownloadURL})
			if len(urls) > 0 {
				video.PlayAddr = &WebPlayAddr{URI: details.ID, URLList: urls, Width: video.Width, Height: video.Height}
			}
		}
		if video.PlayAddr != nil && len(validTikTokURLs(video.PlayAddr.URLList)) > 0 {
			item.AddFormats(&models.MediaFormat{
				Type:       database.MediaTypeVideo,
				FormatID:   video.PlayAddr.URI,
				URL:        validTikTokURLs(video.PlayAddr.URLList),
				VideoCodec: database.MediaCodecAvc,
				AudioCodec: database.MediaCodecAac,
				Width:      video.PlayAddr.Width,
				Height:     video.PlayAddr.Height,
				Duration:   video.Duration,
				DownloadSettings: &models.DownloadSettings{
					// avoid 403 error for videos
					Headers: publicTikTokHeaders(),
					Cookies: publicTikTokCookies(cookies),
				},
			})
			return media, nil
		}
		return nil, util.ErrUnavailable
	} else {
		images := details.ImagePost.Images
		if len(images) > 30 {
			return nil, util.ErrMediaAlbumGlobalLimitExceeded
		}
		for _, image := range images {
			if image == nil || image.URL == nil || len(validTikTokURLs(image.URL.URLList)) == 0 {
				return nil, util.ErrUnavailable
			}
			item := media.NewItem()
			item.AddFormats(&models.MediaFormat{
				Type:     database.MediaTypePhoto,
				FormatID: "image",
				URL:      validTikTokURLs(image.URL.URLList),
			})
		}
		if len(media.Items) == 0 {
			return nil, util.ErrUnavailable
		}
		return media, nil
	}
}
