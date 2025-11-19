package twitter

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/logger"
)

const (
	apiHostname = "x.com"
	apiBase     = "https://" + apiHostname + "/i/api/graphql/"
	apiEndpoint = apiBase + "2ICDjqPd81tulZcYrtpTuQ/TweetResultByRestId"
)

var ShortExtractor = &models.Extractor{
	ID:          "twitter",
	DisplayName: "Twitter (Short)",

	URLPattern: regexp.MustCompile(`https?://t\.co/(?P<id>\w+)`),
	Host:       []string{"t"},

	Redirect: true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		resp, err := ctx.Fetch(
			http.MethodGet,
			ctx.ContentURL,
			nil,
		)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		matchedURL := Extractor.URLPattern.FindSubmatch(body)
		if matchedURL == nil {
			return nil, fmt.Errorf("failed to find tweet url in page")
		}
		return &models.ExtractorResponse{
			URL: string(matchedURL[0]),
		}, nil
	},
}

var Extractor = &models.Extractor{
	ID:          "twitter",
	DisplayName: "Twitter (X)",

	URLPattern: regexp.MustCompile(`https?:\/\/(?:fx|vx|fixup)?(twitter|x)\.com\/([^\/]+)\/status\/(?P<id>\d+)`),
	Host: []string{
		"x",
		"twitter",
		"fxtwitter",
		"vxtwitter",
		"fixuptwitter",
		"fixupx",
	},

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := MediaFromAPI(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			Media: media,
		}, nil
	},
}

func MediaFromAPI(ctx *models.ExtractorContext) (*models.Media, error) {
	media := ctx.NewMedia()

	tweetData, err := GetTweetAPI(ctx)
	if err != nil {
		return nil, err
	}

	caption := SanitizeCaption(tweetData.FullText)
	media.SetCaption(caption)

	var mediaEntities []*MediaEntity
	switch {
	case tweetData.Entities != nil && len(tweetData.Entities.Media) > 0:
		mediaEntities = tweetData.Entities.Media
	case tweetData.ExtendedEntities != nil && len(tweetData.ExtendedEntities.Media) > 0:
		mediaEntities = tweetData.ExtendedEntities.Media
	default:
		return nil, nil
	}

	for _, mediaEntity := range mediaEntities {
		item := media.NewItem()

		switch mediaEntity.Type {
		case "video", "animated_gif":
			formats, err := ExtractVideoFormats(mediaEntity)
			if err != nil {
				return nil, err
			}
			item.AddFormats(formats...)
		case "photo":
			item.AddFormats(&models.MediaFormat{
				Type:     database.MediaTypePhoto,
				FormatID: "photo",
				URL:      []string{mediaEntity.MediaURLHTTPS},
			})
		}
	}

	if len(media.Items) == 0 {
		// tweet has no media
		return nil, nil
	}

	return media, nil
}

func GetTweetAPI(ctx *models.ExtractorContext) (*Tweet, error) {
	tweetID := ctx.ContentID
	if ctx.HTTPClient.Cookies == nil {
		return nil, fmt.Errorf("auth cookies are required")
	}
	headers := BuildAPIHeaders(ctx.HTTPClient.Cookies)
	if headers == nil {
		return nil, fmt.Errorf("invalid auth cookies")
	}
	query := BuildAPIQuery(tweetID)

	reqURL := apiEndpoint + "?" + query
	resp, err := ctx.Fetch(
		http.MethodGet,
		reqURL, &networking.RequestParams{
			Headers: headers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("twitter_api_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code: %s", resp.Status)
	}

	var apiResponse APIResponse
	err = sonic.ConfigFastest.NewDecoder(resp.Body).Decode(&apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	result := apiResponse.Data.TweetResult.Result
	if result == nil {
		return nil, fmt.Errorf("tweet not found")
	}

	if result.TypeName == "TweetUnavailable" {
		return nil, util.ErrUnavailable
	}

	var tweet *Tweet
	switch {
	case result.Tweet != nil:
		tweet = result.Tweet.Legacy
	case result.Legacy != nil:
		tweet = result.Legacy
	default:
		return nil, fmt.Errorf("tweet data not found")
	}

	return tweet, nil
}
