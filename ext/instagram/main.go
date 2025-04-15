package instagram

import (
	"fmt"
	"govd/enums"
	"govd/models"
	"govd/util"
	"io"
	"net/http"
	"regexp"
)

// as a public service, we can't use the official API
// so we use igram.world API, a third-party service
// that provides a similar functionality
// feel free to open PR, if you want to
// add support for the official Instagram API

var httpSession = util.GetHTTPSession()

const (
	apiHostname  = "api.igram.world"
	apiKey       = "aaeaf2805cea6abef3f9d2b6a666fce62fd9d612a43ab772bb50ce81455112e0"
	apiTimestamp = "1742201548873"

	// todo: Implement a proper way
	// to get the API key and timestamp
)

var igHeaders = map[string]string{
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
	"Accept-Language":           "en-GB,en;q=0.9",
	"Cache-Control":             "max-age=0",
	"Dnt":                       "1",
	"Priority":                  "u=0, i",
	"Sec-Ch-Ua":                 `Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99`,
	"Sec-Ch-Ua-Mobile":          "?0",
	"Sec-Ch-Ua-Platform":        "macOS",
	"Sec-Fetch-Dest":            "document",
	"Sec-Fetch-Mode":            "navigate",
	"Sec-Fetch-Site":            "none",
	"Sec-Fetch-User":            "?1",
	"Upgrade-Insecure-Requests": "1",
	"User-Agent":                util.ChromeUA,
}

var Extractor = &models.Extractor{
	Name:       "Instagram",
	CodeName:   "instagram",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/www\.instagram\.com\/(reel|p|tv)\/(?P<id>[a-zA-Z0-9_-]+)`),
	IsRedirect: false,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		mediaList, err := MediaListFromAPI(ctx, false)
		return &models.ExtractorResponse{
			MediaList: mediaList,
		}, err
	},
}

var StoriesExtractor = &models.Extractor{
	Name:       "Instagram Stories",
	CodeName:   "instagram:stories",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https:\/\/www\.instagram\.com\/stories\/[a-zA-Z0-9._]+\/(?P<id>\d+)`),
	IsRedirect: false,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		mediaList, err := MediaListFromAPI(ctx, true)
		return &models.ExtractorResponse{
			MediaList: mediaList,
		}, err
	},
}

var ShareURLExtractor = &models.Extractor{
	Name:       "Instagram Share URL",
	CodeName:   "instagram:share",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?:\/\/(www\.)?instagram\.com\/share\/((reels?|video|s|p)\/)?(?P<id>[^\/\?]+)`),
	IsRedirect: true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		req, err := http.NewRequest(
			http.MethodGet,
			ctx.MatchedContentURL,
			nil,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		for k, v := range igHeaders {
			req.Header.Set(k, v)
		}
		resp, err := httpSession.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()
		return &models.ExtractorResponse{
			URL: resp.Request.URL.String(),
		}, nil
	},
}

func MediaListFromAPI(
	ctx *models.DownloadContext,
	stories bool,
) ([]*models.Media, error) {
	var mediaList []*models.Media
	postURL := ctx.MatchedContentURL
	details, err := GetVideoAPI(postURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}
	var caption string
	if !stories {
		// caption, err = GetPostCaption(postURL)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to get caption: %w", err)
		// }

		// todo: fix this (429 error)
	}
	for _, item := range details.Items {
		media := ctx.Extractor.NewMedia(
			ctx.MatchedContentID,
			ctx.MatchedContentURL,
		)
		media.SetCaption(caption)
		urlObj := item.URL[0]
		contentURL, err := GetCDNURL(urlObj.URL)
		if err != nil {
			return nil, err
		}
		thumbnailURL, err := GetCDNURL(item.Thumb)
		if err != nil {
			return nil, err
		}
		fileExt := urlObj.Ext
		formatID := urlObj.Type
		switch fileExt {
		case "mp4":
			media.AddFormat(&models.MediaFormat{
				Type:       enums.MediaTypeVideo,
				FormatID:   formatID,
				URL:        []string{contentURL},
				VideoCodec: enums.MediaCodecAVC,
				AudioCodec: enums.MediaCodecAAC,
				Thumbnail:  []string{thumbnailURL},
			},
			)
		case "jpg", "webp", "heic", "jpeg":
			media.AddFormat(&models.MediaFormat{
				Type:     enums.MediaTypePhoto,
				FormatID: formatID,
				URL:      []string{contentURL},
			})
		default:
			return nil, fmt.Errorf("unknown format: %s", fileExt)
		}
		mediaList = append(mediaList, media)
	}

	return mediaList, nil
}

func GetVideoAPI(contentURL string) (*IGramResponse, error) {
	apiURL := fmt.Sprintf(
		"https://%s/api/convert",
		apiHostname,
	)
	payload, err := BuildSignedPayload(contentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build signed payload: %w", err)
	}
	req, err := http.NewRequest("POST", apiURL, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", util.ChromeUA)

	resp, err := httpSession.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	response, err := ParseIGramResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return response, nil
}
