package reddit

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

var baseHost = []string{"reddit", "redditmedia.com"}

var ShortExtractor = &models.Extractor{
	ID:          "reddit",
	DisplayName: "Reddit (Short)",

	URLPattern: regexp.MustCompile(`https?://(?P<host>(?:\w+\.)?reddit(?:media)?\.com)/(?P<slug>(?:(?:r|user)/[^/]+/)?s/(?P<id>[^/?#&]+))`),
	Host:       baseHost,

	Redirect: true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		resp, err := ctx.Fetch(
			http.MethodGet,
			ctx.ContentURL,
			nil,
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
	ID:          "reddit",
	DisplayName: "Reddit",

	URLPattern: regexp.MustCompile(`https?://(?P<host>(?:\w+\.)?reddit(?:media)?\.com)/(?P<slug>(?:(?:r|user)/[^/]+/)?comments/(?P<id>[^/?#&]+))`),
	Host:       baseHost,

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

// MediaFromAPI implements yt-dlp pure method:
// - title + selftext + thumbnails + crosspost + fallback + dash/hls + gallery
func MediaFromAPI(ctx *models.ExtractorContext) (*models.Media, error) {
	host := ctx.MatchGroups["host"]
	slug := ctx.MatchGroups["slug"]

	manifest, err := GetRedditData(ctx, host, slug, false)
	if err != nil {
		return nil, err
	}

	if len(manifest) == 0 || len(manifest[0].Data.Children) == 0 {
		return nil, fmt.Errorf("no data found in reddit response")
	}

	data := manifest[0].Data.Children[0].Data
	title := data.Title
	isNsfw := data.Over18

	media := ctx.NewMedia()
	if isNsfw {
		media.SetNSFW()
	}

	// yt-dlp: title truncated 72 + selftext as description
	caption := title
	if data.Selftext != "" && data.Selftext != title {
		if caption != "" {
			caption = caption + "\n\n" + data.Selftext
		} else {
			caption = data.Selftext
		}
	}
	media.SetCaption(caption)

	// Gallery handling - yt-dlp style media_metadata RedditVideo + Image
	if len(data.GalleryData.Items) > 0 {
		for _, gItem := range data.GalleryData.Items {
			metaID := gItem.MediaID
			if meta, ok := data.MediaMetadata[metaID]; ok {
				item := media.NewItem()
				switch meta.Type {
				case "Image":
					item.AddFormats(&models.MediaFormat{
						FormatID: "photo",
						Type:     database.MediaTypePhoto,
						URL:      []string{util.UnescapeURL(meta.Media.URL)},
					})
				case "AnimatedImage", "RedditVideo":
					if meta.Media.MP4 != "" {
						item.AddFormats(&models.MediaFormat{
							FormatID:   "video",
							Type:       database.MediaTypeVideo,
							VideoCodec: database.MediaCodecAvc,
							AudioCodec: database.MediaCodecAac,
							URL:        []string{util.UnescapeURL(meta.Media.MP4)},
						})
					}
					if meta.Media.HLSURL != "" {
						fmts, _ := GetHLSFormatsFromURL(ctx, meta.Media.HLSURL, 0)
						item.AddFormats(fmts...)
					}
					if meta.Media.DashURL != "" {
						fmts, _ := GetDASHFormatsFromURL(ctx, meta.Media.DashURL, 0)
						item.AddFormats(fmts...)
					}
					if len(item.Formats) == 0 && meta.Media.URL != "" {
						item.AddFormats(&models.MediaFormat{
							FormatID: "photo",
							Type:     database.MediaTypePhoto,
							URL:      []string{util.UnescapeURL(meta.Media.URL)},
						})
					}
				}
				if len(item.Formats) > 0 {
					media.Items = append(media.Items, item)
				}
			}
		}
		if len(media.Items) > 0 {
			return media, nil
		}
	}

	// Fallback unordered media_metadata (yt-dlp playlist from media_metadata)
	if len(data.MediaMetadata) > 0 {
		var galleryItems []*models.MediaItem
		for _, obj := range data.MediaMetadata {
			if obj.Type == "RedditVideo" {
				item := media.NewItem()
				if obj.Media.HLSURL != "" {
					fmts, _ := GetHLSFormatsFromURL(ctx, obj.Media.HLSURL, 0)
					item.AddFormats(fmts...)
				}
				if obj.Media.DashURL != "" {
					fmts, _ := GetDASHFormatsFromURL(ctx, obj.Media.DashURL, 0)
					item.AddFormats(fmts...)
				}
				if len(item.Formats) == 0 && obj.Media.MP4 != "" {
					item.AddFormats(&models.MediaFormat{
						FormatID:   "video",
						Type:       database.MediaTypeVideo,
						VideoCodec: database.MediaCodecAvc,
						AudioCodec: database.MediaCodecAac,
						URL:        []string{util.UnescapeURL(obj.Media.MP4)},
					})
				}
				if len(item.Formats) > 0 {
					galleryItems = append(galleryItems, item)
				}
			}
		}
		if len(galleryItems) > 0 {
			media.Items = append(media.Items, galleryItems...)
			return media, nil
		}

		// Image gallery
		for _, obj := range data.MediaMetadata {
			var item *models.MediaItem
			switch obj.Type {
			case "Image":
				item = media.NewItem()
				item.AddFormats(&models.MediaFormat{
					FormatID: "photo",
					Type:     database.MediaTypePhoto,
					URL:      []string{util.UnescapeURL(obj.Media.URL)},
				})
			case "AnimatedImage":
				item = media.NewItem()
				item.AddFormats(&models.MediaFormat{
					FormatID:   "video",
					Type:       database.MediaTypeVideo,
					VideoCodec: database.MediaCodecAvc,
					AudioCodec: database.MediaCodecAac,
					URL:        []string{util.UnescapeURL(obj.Media.MP4)},
				})
			}
			if item != nil && len(item.Formats) > 0 {
				media.Items = append(media.Items, item)
			}
		}
		if len(media.Items) > 0 {
			return media, nil
		}
	}

	if !data.IsVideo {
		// Single photo / gif / video preview
		if data.Preview != nil && len(data.Preview.Images) > 0 {
			image := data.Preview.Images[0]
			if data.Preview.VideoPreview != nil {
				item := media.NewItem()
				formats, err := GetHLSFormats(ctx, data.Preview.VideoPreview.FallbackURL, data.Preview.VideoPreview.Duration)
				if err != nil {
					return nil, err
				}
				item.AddFormats(formats...)
				return media, nil
			}
			if image.Variants.MP4 != nil {
				item := media.NewItem()
				item.AddFormats(&models.MediaFormat{
					FormatID:   "gif",
					Type:       database.MediaTypeVideo,
					VideoCodec: database.MediaCodecAvc,
					AudioCodec: database.MediaCodecAac,
					URL:        []string{util.UnescapeURL(image.Variants.MP4.Source.URL)},
				})
				return media, nil
			}
			item := media.NewItem()
			item.AddFormats(&models.MediaFormat{
				FormatID: "photo",
				Type:     database.MediaTypePhoto,
				URL:      []string{util.UnescapeURL(image.Source.URL)},
				Width:    image.Source.Width,
				Height:   image.Source.Height,
			})
			return media, nil
		}

		// Crosspost check (yt-dlp style)
		if len(data.CrosspostParentList) > 0 {
			for _, parent := range data.CrosspostParentList {
				var rv *Video
				if parent.Media != nil && parent.Media.Video != nil {
					rv = parent.Media.Video
				} else if parent.SecureMedia != nil && parent.SecureMedia.Video != nil {
					rv = parent.SecureMedia.Video
				}
				if rv != nil {
					item := media.NewItem()
					formats := buildRedditVideoFormats(ctx, rv)
					item.AddFormats(formats...)
					if len(item.Formats) > 0 {
						return media, nil
					}
				}
			}
		}
	} else {
		var redditVideo *Video
		if data.Media != nil && data.Media.Video != nil {
			redditVideo = data.Media.Video
		} else if data.SecureMedia != nil && data.SecureMedia.Video != nil {
			redditVideo = data.SecureMedia.Video
		}
		if redditVideo == nil && len(data.CrosspostParentList) > 0 {
			for _, parent := range data.CrosspostParentList {
				if parent.Media != nil && parent.Media.Video != nil {
					redditVideo = parent.Media.Video
					break
				} else if parent.SecureMedia != nil && parent.SecureMedia.Video != nil {
					redditVideo = parent.SecureMedia.Video
					break
				}
			}
		}
		if redditVideo != nil {
			item := media.NewItem()
			formats := buildRedditVideoFormats(ctx, redditVideo)
			item.AddFormats(formats...)
			if len(item.Formats) > 0 {
				return media, nil
			}
		}
	}

	// Fallback preview video
	if data.Preview != nil && data.Preview.VideoPreview != nil {
		item := media.NewItem()
		formats, err := GetHLSFormats(ctx, data.Preview.VideoPreview.FallbackURL, data.Preview.VideoPreview.Duration)
		if err == nil {
			item.AddFormats(formats...)
			return media, nil
		}
	}

	return nil, nil
}

// buildRedditVideoFormats implements yt-dlp reddit_video handling: fallback + hls + dash
func buildRedditVideoFormats(ctx *models.ExtractorContext, rv *Video) []*models.MediaFormat {
	var formats []*models.MediaFormat

	if rv.FallbackURL != "" {
		formats = append(formats, &models.MediaFormat{
			FormatID:   "fallback",
			Type:       database.MediaTypeVideo,
			VideoCodec: database.MediaCodecAvc,
			AudioCodec: database.MediaCodecAac,
			URL:        []string{util.UnescapeURL(rv.FallbackURL)},
			Width:      rv.Width,
			Height:     rv.Height,
			Duration:   rv.Duration,
		})
	}

	if rv.HLSURL != "" {
		hlsFmts, _ := GetHLSFormatsFromURL(ctx, rv.HLSURL, rv.Duration)
		formats = append(formats, hlsFmts...)
	} else if rv.FallbackURL != "" {
		if m := videoURLPattern.FindStringSubmatch(rv.FallbackURL); len(m) >= 2 {
			videoID := m[1]
			hlsURL := fmt.Sprintf(hlsURLFormat, videoID)
			hlsFmts, _ := GetHLSFormatsFromURL(ctx, hlsURL, rv.Duration)
			formats = append(formats, hlsFmts...)
		}
	}

	if rv.DashURL != "" {
		dashFmts, _ := GetDASHFormatsFromURL(ctx, rv.DashURL, rv.Duration)
		formats = append(formats, dashFmts...)
	} else if rv.FallbackURL != "" {
		if m := videoURLPattern.FindStringSubmatch(rv.FallbackURL); len(m) >= 2 {
			videoID := m[1]
			dashURL := fmt.Sprintf("https://v.redd.it/%s/DASHPlaylist.mpd", videoID)
			dashFmts, _ := GetDASHFormatsFromURL(ctx, dashURL, rv.Duration)
			formats = append(formats, dashFmts...)
		}
	}

	if len(formats) == 0 && rv.FallbackURL != "" {
		legacy, _ := GetHLSFormats(ctx, rv.FallbackURL, rv.Duration)
		formats = append(formats, legacy...)
	}

	return formats
}

func GetRedditData(
	ctx *models.ExtractorContext,
	host string,
	slug string,
	raise bool,
) (Response, error) {
	url := fmt.Sprintf("https://%s/%s/.json", host, slug)

	resp, err := ctx.Fetch(
		http.MethodGet,
		url, nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if raise {
			return nil, fmt.Errorf("failed to get reddit data: %s", resp.Status)
		}
		altHost := "old.reddit.com"
		if host == "old.reddit.com" {
			altHost = "www.reddit.com"
		}
		return GetRedditData(ctx, altHost, slug, true)
	}

	logger.WriteFile("reddit_api_response", resp)

	var response Response
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return response, nil
}
