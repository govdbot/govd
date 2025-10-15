package pinterest

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"

	"github.com/bytedance/sonic"
)

const (
	pinResourceEndpoint = "https://www.pinterest.com/resource/PinResource/get/"
	shortenerAPIFormat  = "https://api.pinterest.com/url_shortener/%s/redirect/"
)

var ShortExtractor = &models.Extractor{
	ID:          "pinterest",
	DisplayName: "Pinterest (Short)",

	URLPattern: regexp.MustCompile(`https?://(www\.)?pin\.[^/]+/(?P<id>\w+)`),
	Host:       []string{"pin"},

	Redirect: true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		shortURL := fmt.Sprintf(shortenerAPIFormat, ctx.ContentID)
		url, err := ctx.FetchLocation(shortURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get real url: %w", err)
		}
		return &models.ExtractorResponse{URL: url}, nil
	},
}

var Extractor = &models.Extractor{
	ID:          "pinterest",
	DisplayName: "Pinterest",

	URLPattern: regexp.MustCompile(`https?://(?:[^/]+\.)?pinterest\.[^/]+/pin/(?:[\w-]+--)?(?P<id>\d+)`),
	Host:       []string{"pinterest"},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := ExtractPinMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{Media: media}, nil
	},
}

func ExtractPinMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	pinData, err := GetPinData(ctx)
	if err != nil {
		return nil, err
	}

	media := ctx.NewMedia()
	media.SetCaption(pinData.Title)

	if pinData.Videos != nil && pinData.Videos.VideoList != nil {
		item := media.NewItem()
		formats, err := ParseVideoObject(ctx, pinData.Videos)
		if err != nil {
			return nil, err
		}
		item.AddFormats(formats...)
		return media, nil
	}

	if pinData.StoryPinData != nil && len(pinData.StoryPinData.Pages) > 0 {
		for _, page := range pinData.StoryPinData.Pages {
			for _, block := range page.Blocks {
				if block.BlockType == 3 && block.Video != nil {
					// blockType 3 = Video
					item := media.NewItem()
					formats, err := ParseVideoObject(ctx, block.Video)
					if err != nil {
						return nil, err
					}
					item.AddFormats(formats...)
					return media, nil
				}
			}
		}
	}

	if pinData.Images != nil && pinData.Images.Orig != nil {
		item := media.NewItem()
		imageURL := pinData.Images.Orig.URL
		item.AddFormats(&models.MediaFormat{
			FormatID: "photo",
			Type:     database.MediaTypePhoto,
			URL:      []string{imageURL},
		})
		return media, nil
	} else if pinData.StoryPinData != nil && len(pinData.StoryPinData.Pages) > 0 {
		for _, page := range pinData.StoryPinData.Pages {
			if page.Image != nil && page.Image.Images.Originals != nil {
				item := media.NewItem()
				item.AddFormats(&models.MediaFormat{
					FormatID: "photo",
					Type:     database.MediaTypePhoto,
					URL:      []string{page.Image.Images.Originals.URL},
				})
				return media, nil
			}
		}
	}

	if pinData.Embed != nil && pinData.Embed.Type == "gif" {
		item := media.NewItem()
		item.AddFormats(&models.MediaFormat{
			FormatID:   "gif",
			Type:       database.MediaTypeVideo,
			VideoCodec: database.MediaCodecAvc,
			URL:        []string{pinData.Embed.Src},
		})
		return media, nil
	}

	return nil, fmt.Errorf("no media found for pin ID: %s", ctx.ContentID)
}

func GetPinData(ctx *models.ExtractorContext) (*PinData, error) {
	params := BuildPinRequestParams(ctx.ContentID)
	reqURL := pinResourceEndpoint + "?" + params

	resp, err := ctx.Fetch(
		http.MethodGet,
		reqURL,
		&networking.RequestParams{
			Headers: headers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("pinterest_api_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response: %s", resp.Status)
	}

	var pinResponse PinResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&pinResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &pinResponse.ResourceResponse.Data, nil
}
