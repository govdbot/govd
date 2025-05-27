package reddit

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

var baseHost = []string{
	"reddit",
	"redditmedia.com",
}

var ShortExtractor = &models.Extractor{
	Name:       "Reddit (Short)",
	CodeName:   "reddit",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(`https?://(?P<host>(?:\w+\.)?reddit(?:media)?\.com)/(?P<slug>(?:(?:r|user)/[^/]+/)?s/(?P<id>[^/?#&]+))`),
	Host:       baseHost,
	IsRedirect: true,
	IsHidden:   true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		client := networking.GetExtractorHTTPClient(ctx.Extractor)
		cookies := util.GetExtractorCookies(ctx.Extractor)
		if cookies == nil {
			return nil, util.ErrAuthenticationNeeded
		}

		resp, err := util.FetchPage(
			client,
			http.MethodGet,
			ctx.MatchedContentURL,
			nil,
			nil,
			cookies,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
		defer resp.Body.Close()

		location := resp.Request.URL.String()

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

	manifest, err := GetRedditData(ctx, host, slug, false)
	if err != nil {
		return nil, err
	}

	if len(manifest) == 0 || len(manifest[0].Data.Children) == 0 {
		return nil, ErrNoDataFound
	}

	data := manifest[0].Data.Children[0].Data
	title := data.Title
	isNsfw := data.Over18

	if !data.IsVideo {
		// check for single photo
		if data.Preview != nil && len(data.Preview.Images) > 0 {
			media := ctx.Extractor.NewMedia(contentID, contentURL)
			media.SetCaption(title)
			media.NSFW = isNsfw

			image := data.Preview.Images[0]

			// check for video preview (GIF)
			if data.Preview.VideoPreview != nil {
				formats, err := GetHLSFormats(
					data.Preview.VideoPreview.FallbackURL,
					image.Source.URL,
					data.Preview.VideoPreview.Duration,
				)
				if err != nil {
					return nil, err
				}

				for _, format := range formats {
					media.AddFormat(format)
				}

				return []*models.Media{media}, nil
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

				return []*models.Media{media}, nil
			}

			// regular photo
			media.AddFormat(&models.MediaFormat{
				FormatID: "photo",
				Type:     enums.MediaTypePhoto,
				URL:      []string{util.FixURL(image.Source.URL)},
			})

			return []*models.Media{media}, nil
		}

		// check for gallery/collection
		if len(data.MediaMetadata) > 0 {
			// known issue: collection is unordered
			collection := data.MediaMetadata
			mediaList := make([]*models.Media, 0, len(collection))

			for _, obj := range collection {
				media := ctx.Extractor.NewMedia(contentID, contentURL)
				media.SetCaption(title)
				media.NSFW = isNsfw

				switch obj.Type {
				case "Image":
					media.AddFormat(&models.MediaFormat{
						FormatID: "photo",
						Type:     enums.MediaTypePhoto,
						URL:      []string{util.FixURL(obj.Media.URL)},
					})
				case "AnimatedImage":
					media.AddFormat(&models.MediaFormat{
						FormatID:   "video",
						Type:       enums.MediaTypeVideo,
						VideoCodec: enums.MediaCodecAVC,
						AudioCodec: enums.MediaCodecAAC,
						URL:        []string{util.FixURL(obj.Media.MP4)},
					})
				}
				mediaList = append(mediaList, media)
			}
			return mediaList, nil
		}
	} else {
		// video
		media := ctx.Extractor.NewMedia(contentID, contentURL)
		media.SetCaption(title)
		media.NSFW = isNsfw

		var redditVideo *Video

		if data.Media != nil && data.Media.Video != nil {
			redditVideo = data.Media.Video
		} else if data.SecureMedia != nil && data.SecureMedia.Video != nil {
			redditVideo = data.SecureMedia.Video
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

			return []*models.Media{media}, nil
		}
	}

	// no media found
	return nil, nil
}

func GetRedditData(
	ctx *models.DownloadContext,
	host string,
	slug string,
	raise bool,
) (Response, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	url := fmt.Sprintf("https://%s/%s/.json", host, slug)

	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		url,
		nil,
		nil,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if raise {
			return nil, fmt.Errorf("failed to get reddit data: %s", resp.Status)
		}
		// try with alternative domain
		altHost := "old.reddit.com"
		if host == "old.reddit.com" {
			altHost = "www.reddit.com"
		}

		return GetRedditData(ctx, altHost, slug, true)
	}

	// debugging
	logger.WriteFile("reddit_api_response", resp)

	var response Response
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}
