package reddit

import (
	"fmt"
	"net/http"
	"regexp"

	"govd/enums"
	"govd/models"
	"govd/util"

	"github.com/bytedance/sonic"
)

var (
	httpSession = util.GetHTTPSession()
	baseHost    = []string{
		"reddit.com",
		"redditmedia.com",
		"old.reddit.com",
		"old.redditmedia.com",
	}
)

var ShortExtractor = &models.Extractor{
	Name:       "Reddit (Short)",
	CodeName:   "reddit:short",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(?P<host>(?:\w+\.)?reddit(?:media)?\.com)/(?P<slug>(?:(?:r|user)/[^/]+/)?s/(?P<id>[^/?#&]+))`),
	Host:       baseHost,
	IsRedirect: true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		req, err := http.NewRequest(http.MethodGet, ctx.MatchedContentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", util.ChromeUA)
		cookies, err := util.ParseCookieFile("reddit.txt")
		if err != nil {
			return nil, fmt.Errorf("failed to get cookies: %w", err)
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}

		res, err := httpSession.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer res.Body.Close()

		location := res.Request.URL.String()

		return &models.ExtractorResponse{
			URL: location,
		}, nil
	},
}

var Extractor = &models.Extractor{
	Name:       "Reddit",
	CodeName:   "reddit",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(?P<host>(?:\w+\.)?reddit(?:media)?\.com)/(?P<slug>(?:(?:r|user)/[^/]+/)?comments/(?P<id>[^/?#&]+))`),
	Host:       baseHost,

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
	host := ctx.MatchedGroups["host"]
	slug := ctx.MatchedGroups["slug"]

	contentID := ctx.MatchedContentID
	contentURL := ctx.MatchedContentURL

	manifest, err := GetRedditData(host, slug)
	if err != nil {
		return nil, err
	}

	if len(manifest) == 0 || len(manifest[0].Data.Children) == 0 {
		return nil, fmt.Errorf("no data found in response")
	}

	data := manifest[0].Data.Children[0].Data
	title := data.Title
	isNsfw := data.Over18
	var mediaList []*models.Media

	if !data.IsVideo {
		// check for single photo
		if data.Preview != nil && len(data.Preview.Images) > 0 {
			media := ctx.Extractor.NewMedia(contentID, contentURL)
			media.SetCaption(title)
			if isNsfw {
				media.NSFW = true
			}

			image := data.Preview.Images[0]

			// check for video preview (GIF)
			if data.Preview.RedditVideoPreview != nil {
				formats, err := GetHLSFormats(
					data.Preview.RedditVideoPreview.FallbackURL,
					image.Source.URL,
					data.Preview.RedditVideoPreview.Duration,
				)
				if err != nil {
					return nil, err
				}

				for _, format := range formats {
					media.AddFormat(format)
				}

				mediaList = append(mediaList, media)
				return mediaList, nil
			}

			// check for MP4 variant (animated GIF)
			if image.Variants.MP4 != nil {
				media.AddFormat(&models.MediaFormat{
					FormatID:   "gif",
					Type:       enums.MediaTypeVideo,
					VideoCodec: enums.MediaCodecAVC,
					AudioCodec: enums.MediaCodecAAC,
					URL:        []string{util.FixURL(image.Variants.MP4.Source.URL)},
					Thumbnail:  []string{util.FixURL(image.Source.URL)},
				})

				mediaList = append(mediaList, media)
				return mediaList, nil
			}

			// regular photo
			media.AddFormat(&models.MediaFormat{
				FormatID: "photo",
				Type:     enums.MediaTypePhoto,
				URL:      []string{util.FixURL(image.Source.URL)},
			})

			mediaList = append(mediaList, media)
			return mediaList, nil
		}

		// check for gallery/collection
		if len(data.MediaMetadata) > 0 {
			for key, obj := range data.MediaMetadata {
				if obj.E == "Image" {
					media := ctx.Extractor.NewMedia(key, contentURL)
					media.SetCaption(title)
					if isNsfw {
						media.NSFW = true
					}

					media.AddFormat(&models.MediaFormat{
						FormatID: "photo",
						Type:     enums.MediaTypePhoto,
						URL:      []string{util.FixURL(obj.S.U)},
					})

					mediaList = append(mediaList, media)
				}
			}

			return mediaList, nil
		}
	} else {
		// video
		media := ctx.Extractor.NewMedia(contentID, contentURL)
		media.SetCaption(title)
		if isNsfw {
			media.NSFW = true
		}

		var redditVideo *RedditVideo

		if data.Media != nil && data.Media.RedditVideo != nil {
			redditVideo = data.Media.RedditVideo
		} else if data.SecureMedia != nil && data.SecureMedia.RedditVideo != nil {
			redditVideo = data.SecureMedia.RedditVideo
		}

		if redditVideo != nil {
			thumbnail := data.Thumbnail

			if (thumbnail == "nsfw" || thumbnail == "spoiler") && data.Preview != nil && len(data.Preview.Images) > 0 {
				thumbnail = data.Preview.Images[0].Source.URL
			}

			formats, err := GetHLSFormats(
				redditVideo.FallbackURL,
				thumbnail,
				redditVideo.Duration,
			)
			if err != nil {
				return nil, err
			}

			for _, format := range formats {
				media.AddFormat(format)
			}

			mediaList = append(mediaList, media)
			return mediaList, nil
		}
	}

	return mediaList, nil
}

func GetRedditData(host string, slug string) (RedditResponse, error) {
	url := fmt.Sprintf("https://%s/%s/.json", host, slug)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", util.ChromeUA)
	cookies, err := util.ParseCookieFile("reddit.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to get cookies: %w", err)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	res, err := httpSession.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		// try with alternative domain
		altHost := "old.reddit.com"
		if host == "old.reddit.com" {
			altHost = "www.reddit.com"
		}

		return GetRedditData(altHost, slug)
	}

	var response RedditResponse
	decoder := sonic.ConfigFastest.NewDecoder(res.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}
