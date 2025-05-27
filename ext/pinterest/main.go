package pinterest

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/logger"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/networking"

	"github.com/bytedance/sonic"
)

const (
	pinResourceEndpoint = "https://www.pinterest.com/resource/PinResource/get/"
	shortenerAPIFormat  = "https://api.pinterest.com/url_shortener/%s/redirect/"
)

var ShortExtractor = &models.Extractor{
	Name:       "Pinterest (Short)",
	CodeName:   "pinterest",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(www\.)?pin\.[^/]+/(?P<id>\w+)`),
	Host:       []string{"pin"},
	IsRedirect: true,
	IsHidden:   true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		client := networking.GetExtractorHTTPClient(ctx.Extractor)
		cookies := util.GetExtractorCookies(ctx.Extractor)
		shortURL := fmt.Sprintf(shortenerAPIFormat, ctx.MatchedContentID)
		location, err := util.GetLocationURL(client, shortURL, nil, cookies)
		if err != nil {
			return nil, fmt.Errorf("failed to get real url: %w", err)
		}
		return &models.ExtractorResponse{
			URL: location,
		}, nil
	},
}

var Extractor = &models.Extractor{
	Name:       "Pinterest",
	CodeName:   "pinterest",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(?:[^/]+\.)?pinterest\.[^/]+/pin/(?:[\w-]+--)?(?P<id>\d+)`),
	Host:       []string{"pinterest"},

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		media, err := ExtractPinMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			MediaList: media,
		}, nil
	},
}

func ExtractPinMedia(ctx *models.DownloadContext) ([]*models.Media, error) {
	pinID := ctx.MatchedContentID
	contentURL := ctx.MatchedContentURL

	pinData, err := GetPinData(ctx, pinID)
	if err != nil {
		return nil, err
	}

	media := ctx.Extractor.NewMedia(pinID, contentURL)
	media.SetCaption(pinData.Title)

	if pinData.Videos != nil && pinData.Videos.VideoList != nil {
		formats, err := ParseVideoObject(pinData.Videos)
		if err != nil {
			return nil, err
		}
		for _, format := range formats {
			media.AddFormat(format)
		}
		return []*models.Media{media}, nil
	}

	if pinData.StoryPinData != nil && len(pinData.StoryPinData.Pages) > 0 {
		for _, page := range pinData.StoryPinData.Pages {
			for _, block := range page.Blocks {
				if block.BlockType == 3 && block.Video != nil { // blockType 3 = Video
					formats, err := ParseVideoObject(block.Video)
					if err != nil {
						return nil, err
					}
					for _, format := range formats {
						media.AddFormat(format)
					}
					return []*models.Media{media}, nil
				}
			}
		}
	}

	if pinData.Images != nil && pinData.Images.Orig != nil {
		imageURL := pinData.Images.Orig.URL
		media.AddFormat(&models.MediaFormat{
			FormatID: "photo",
			Type:     enums.MediaTypePhoto,
			URL:      []string{imageURL},
		})
		return []*models.Media{media}, nil
	} else if pinData.StoryPinData != nil && len(pinData.StoryPinData.Pages) > 0 {
		for _, page := range pinData.StoryPinData.Pages {
			if page.Image != nil && page.Image.Images.Originals != nil {
				media.AddFormat(&models.MediaFormat{
					FormatID: "photo",
					Type:     enums.MediaTypePhoto,
					URL:      []string{page.Image.Images.Originals.URL},
				})
				return []*models.Media{media}, nil
			}
		}
	}

	if pinData.Embed != nil && pinData.Embed.Type == "gif" {
		media.AddFormat(&models.MediaFormat{
			FormatID:   "gif",
			Type:       enums.MediaTypeVideo,
			VideoCodec: enums.MediaCodecAVC,
			URL:        []string{pinData.Embed.Src},
		})
		return []*models.Media{media}, nil
	}

	return nil, fmt.Errorf("no media found for pin ID: %s", pinID)
}

func GetPinData(
	ctx *models.DownloadContext,
	pinID string,
) (*PinData, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)
	params := BuildPinRequestParams(pinID)
	reqURL := pinResourceEndpoint + "?" + params

	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		reqURL,
		nil,
		headers,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
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
