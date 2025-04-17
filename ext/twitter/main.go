package twitter

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"govd/enums"
	"govd/models"
	"govd/util"

	"github.com/bytedance/sonic"
)

const (
	apiHostname = "x.com"
	apiEndpoint = "https://x.com/i/api/graphql/zZXycP0V6H7m-2r0mOnFcA/TweetDetail"
)

var httpSession = util.GetHTTPSession()

var ShortExtractor = &models.Extractor{
	Name:       "Twitter (Short)",
	CodeName:   "twitter:short",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://t\.co/(?P<id>\w+)`),
	Host:       []string{"t.co"},
	IsRedirect: true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		req, err := http.NewRequest(http.MethodGet, ctx.MatchedContentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create req: %w", err)
		}
		req.Header.Set("User-Agent", util.ChromeUA)
		res, err := httpSession.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer res.Body.Close()
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read body: %w", err)
		}
		matchedURL := Extractor.URLPattern.FindStringSubmatch(string(body))
		if matchedURL == nil {
			return nil, fmt.Errorf("failed to find url in body")
		}
		return &models.ExtractorResponse{
			URL: matchedURL[0],
		}, nil
	},
}

var Extractor = &models.Extractor{
	Name:       "Twitter",
	CodeName:   "twitter",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?:\/\/(vx)?(twitter|x)\.com\/([^\/]+)\/status\/(?P<id>\d+)`),
	Host: []string{
		"twitter.com",
		"x.com",
		"vxx.com",
		"vxtwitter.com",
	},

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
	var mediaList []*models.Media

	tweetData, err := GetTweetAPI(ctx.MatchedContentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tweet data: %w", err)
	}

	caption := CleanCaption(tweetData.FullText)

	var mediaEntities []MediaEntity
	if tweetData.ExtendedEntities != nil && len(tweetData.ExtendedEntities.Media) > 0 {
		mediaEntities = tweetData.ExtendedEntities.Media
	} else if tweetData.Entities != nil && len(tweetData.Entities.Media) > 0 {
		mediaEntities = tweetData.Entities.Media
	} else {
		return nil, fmt.Errorf("no media found in tweet")
	}

	for _, mediaEntity := range mediaEntities {
		media := ctx.Extractor.NewMedia(
			ctx.MatchedContentID,
			ctx.MatchedContentURL,
		)
		media.SetCaption(caption)

		switch mediaEntity.Type {
		case "video", "animated_gif":
			formats, err := ExtractVideoFormats(&mediaEntity)
			if err != nil {
				return nil, err
			}
			for _, format := range formats {
				media.AddFormat(format)
			}
		case "photo":
			media.AddFormat(&models.MediaFormat{
				Type:     enums.MediaTypePhoto,
				FormatID: "photo",
				URL:      []string{mediaEntity.MediaURLHTTPS},
			})
		}

		if len(media.Formats) > 0 {
			mediaList = append(mediaList, media)
		}
	}

	return mediaList, nil
}

func GetTweetAPI(tweetID string) (*Tweet, error) {
	cookies, err := util.ParseCookieFile("twitter.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}
	headers := BuildAPIHeaders(cookies)
	if headers == nil {
		return nil, fmt.Errorf("failed to build headers. check cookies")
	}
	query := BuildAPIQuery(tweetID)

	req, err := http.NewRequest(http.MethodGet, apiEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create req: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	q := req.URL.Query()
	for key, value := range query {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()

	resp, err := httpSession.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code: %s", resp.Status)
	}

	var apiResponse APIResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&apiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	tweet, err := FindTweetData(&apiResponse, tweetID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tweet data: %w", err)
	}

	return tweet, nil
}
