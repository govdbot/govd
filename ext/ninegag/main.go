package ninegag

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
	apiEndpoint  = "https://9gag.com/v1/post"
	postNotFound = "Post not found"
)

// 9gag gives 403 unless you use
// real browser TLS fingerprint
var httpClient = networking.NewChromeClient()

var Extractor = &models.Extractor{
	Name:       "9GAG",
	CodeName:   "ninegag",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(?:www\.)?9gag\.com/gag/(?P<id>[^/?&#]+)`),
	Host:       []string{"9gag"},

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		mediaList, err := MediaListFromAPI(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get media: %w", err)
		}
		return &models.ExtractorResponse{
			MediaList: mediaList,
		}, nil
	},
}

func MediaListFromAPI(ctx *models.DownloadContext) ([]*models.Media, error) {
	contentID := ctx.MatchedContentID
	contentURL := ctx.MatchedContentURL

	postData, err := GetPostData(contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get post data: %w", err)
	}

	media := ctx.Extractor.NewMedia(contentID, contentURL)
	media.SetCaption(postData.Title)

	if postData.Nsfw == 1 {
		media.NSFW = true
	}

	switch postData.Type {
	case "Photo":
		bestPhoto, err := FindBestPhoto(postData.Images)
		if err != nil {
			return nil, err
		}

		media.AddFormat(&models.MediaFormat{
			FormatID: "photo",
			Type:     enums.MediaTypePhoto,
			URL:      []string{bestPhoto.URL},
			Width:    int64(bestPhoto.Width),
			Height:   int64(bestPhoto.Height),
		})

	case "Animated":
		videoFormats, err := ParseVideoFormats(postData.Images)
		if err != nil {
			return nil, err
		}

		for _, format := range videoFormats {
			media.AddFormat(format)
		}
	}
	if len(media.Formats) > 0 {
		return []*models.Media{media}, nil
	}

	// no media found
	return nil, nil
}

func GetPostData(postID string) (*Post, error) {
	url := apiEndpoint + "?id=" + postID
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", util.ChromeUA)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("9gag_api_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	var response Response
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Meta != nil && response.Meta.Status != "Success" {
		return nil, fmt.Errorf("API error: %s", response.Meta.Status)
	}

	if response.Meta != nil && response.Meta.ErrorMessage == postNotFound {
		return nil, util.ErrUnavailable
	}

	if response.Data == nil || response.Data.Post == nil {
		return nil, ErrNoMediaFound
	}

	return response.Data.Post, nil
}
