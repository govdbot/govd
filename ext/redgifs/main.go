package redgifs

import (
	"fmt"
	"maps"
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
	baseAPI       = "https://api.redgifs.com/v2/"
	tokenEndpoint = baseAPI + "auth/temporary"
	videoEndpoint = baseAPI + "gifs/"
)

var baseAPIHeaders = map[string]string{
	"Referer":      "https://www.redgifs.com/",
	"Origin":       "https://www.redgifs.com",
	"Content-Type": "application/json",
}

var Extractor = &models.Extractor{
	Name:       "RedGifs",
	CodeName:   "redgifs",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(?:(?:www\.)?redgifs\.com/(?:watch|ifr)/|thumbs2\.redgifs\.com/)(?P<id>[^-/?#\.]+)`),
	Host:       []string{"redgifs"},

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
	response, err := GetVideo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get from api: %w", err)
	}
	gif := response.Gif
	media := ctx.Extractor.NewMedia(
		ctx.MatchedContentID,
		ctx.MatchedContentURL,
	)

	if gif.Description != "" {
		media.SetCaption(gif.Description)
	}
	media.NSFW = true // always nsfw

	if gif.Urls.Sd != "" {
		format := &models.MediaFormat{
			FormatID:   "sd",
			Type:       enums.MediaTypeVideo,
			URL:        []string{gif.Urls.Sd},
			VideoCodec: enums.MediaCodecAVC,
			Width:      int64(gif.Width / 2),
			Height:     int64(gif.Height / 2),
		}
		if gif.HasAudio {
			format.AudioCodec = enums.MediaCodecAAC
		}
		media.AddFormat(format)
	}

	if gif.Urls.Hd != "" {
		format := &models.MediaFormat{
			FormatID:   "hd",
			Type:       enums.MediaTypeVideo,
			URL:        []string{gif.Urls.Hd},
			VideoCodec: enums.MediaCodecAVC,
			Width:      int64(gif.Width),
			Height:     int64(gif.Height),
		}
		if gif.HasAudio {
			format.AudioCodec = enums.MediaCodecAAC
		}
		media.AddFormat(format)
	}

	if gif.Urls.Poster != "" {
		thumbnails := []string{gif.Urls.Poster}
		if gif.Urls.Thumbnail != "" {
			thumbnails = append(thumbnails, gif.Urls.Thumbnail)
		}

		for _, format := range media.Formats {
			format.Thumbnail = thumbnails
			format.Duration = int64(gif.Duration)
		}
	}

	return []*models.Media{media}, nil
}

func GetVideo(ctx *models.DownloadContext) (*Response, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	videoID := ctx.MatchedContentID
	url := videoEndpoint + videoID + "?views=true"
	token, err := GetAccessToken(client, cookies)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	headers := map[string]string{
		"User-Agent":     token.Agent,
		"Authorization":  "Bearer " + token.AccessToken,
		"X-Customheader": "https://www.redgifs.com/watch/" + videoID,
	}
	maps.Copy(headers, baseAPIHeaders)
	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		url,
		nil,
		headers,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("redgifs_api_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response: %s", resp.Status)
	}

	var response Response
	err = sonic.ConfigFastest.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if response.Gif == nil {
		return nil, fmt.Errorf("failed to get video: %s", resp.Status)
	}
	return &response, nil
}
