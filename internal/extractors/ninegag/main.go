package ninegag

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"

	"github.com/bytedance/sonic"
)

const (
	apiEndpoint  = "https://9gag.com/v1/post"
	postNotFound = "Post not found"
)

var Extractor = &models.Extractor{
	ID:          "ninegag",
	DisplayName: "9GAG",

	URLPattern: regexp.MustCompile(`https?://(?:www\.)?9gag\.com/gag/(?P<id>[^/?&#]+)`),
	Host:       []string{"9gag"},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := MediaFromAPI(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get media: %w", err)
		}
		return &models.ExtractorResponse{
			Media: media,
		}, nil
	},
}

func MediaFromAPI(ctx *models.ExtractorContext) (*models.Media, error) {
	postData, err := GetPostData(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get post data: %w", err)
	}
	media := ctx.NewMedia()
	media.SetCaption(postData.Title)

	if postData.Nsfw == 1 {
		media.NSFW = true
	}

	item := media.NewItem()

	switch postData.Type {
	case "Photo":
		bestPhoto, err := FindBestPhoto(postData.Images)
		if err != nil {
			return nil, err
		}
		item.AddFormats(&models.MediaFormat{
			FormatID: "photo",
			Type:     database.MediaTypePhoto,
			URL:      []string{bestPhoto.URL},
			Width:    bestPhoto.Width,
			Height:   bestPhoto.Height,
		})
	case "Animated":
		videoFormats, err := ParseVideoFormats(postData.Images)
		if err != nil {
			return nil, err
		}
		item.AddFormats(videoFormats...)
	}
	if len(media.Items) > 0 {
		return media, nil
	}

	// no media found
	return nil, nil
}

func GetPostData(ctx *models.ExtractorContext) (*Post, error) {
	url := apiEndpoint + "?id=" + ctx.ContentID

	resp, err := ctx.Fetch(
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

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
		return nil, fmt.Errorf("no post data found in response")
	}

	return response.Data.Post, nil
}
