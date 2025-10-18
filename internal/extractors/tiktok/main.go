package tiktok

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

const (
	apiEndpoint = "https://www.tiktok.com/player/api/v1/items"
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
	data, err := GetFromWebAPI(ctx)
	if err != nil {
		return nil, err
	}
	media := ctx.NewMedia()
	media.SetCaption(data.Desc)

	videoInfo := data.VideoInfo
	if videoInfo == nil {
		return nil, fmt.Errorf("no video info found")
	}

	duration := int32(videoInfo.Meta.Duration / 1000)

	isSlide := data.ImagePostInfo != nil && len(data.ImagePostInfo.Images) > 0

	if !isSlide {
		videoInfo := data.VideoInfo

		item := media.NewItem()
		item.AddFormats(&models.MediaFormat{
			FormatID:   data.VideoInfo.URI,
			Type:       database.MediaTypeVideo,
			URL:        videoInfo.URLList,
			Duration:   duration,
			Width:      int32(videoInfo.Meta.Width),
			Height:     int32(videoInfo.Meta.Height),
			Bitrate:    int32(videoInfo.Meta.Bitrate),
			AudioCodec: database.MediaCodecAac,
			VideoCodec: database.MediaCodecAvc,
		})

		return media, nil
	}

	for i, image := range data.ImagePostInfo.Images {
		item := media.NewItem()
		item.AddFormats(&models.MediaFormat{
			FormatID: "image" + strconv.Itoa(i),
			Type:     database.MediaTypePhoto,
			URL:      image.DisplayImage.URLList,
		})
	}

	return media, nil
}

func GetFromWebAPI(ctx *models.ExtractorContext) (*Item, error) {
	videoID := ctx.ContentID
	apiURL := apiEndpoint + "?" + RequestParams(videoID).Encode()
	resp, err := ctx.Fetch(
		http.MethodGet,
		apiURL,
		nil,
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	logger.WriteFile("tiktok_api", resp)

	var data *Response
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if data.StatusCode == 2053 {
		return nil, util.ErrUnavailable
	}
	if len(data.Items) == 0 {
		return nil, fmt.Errorf("no items found in response")
	}
	var item *Item
	for i := range data.Items {
		detail := data.Items[i]
		if detail.IDStr == videoID {
			item = detail
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("no matching item found in response")
	}
	return item, nil
}
