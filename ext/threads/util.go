package threads

import (
	"bytes"
	"fmt"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"

	"github.com/PuerkitoBio/goquery"
)

var headers = map[string]string{
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
}

func ParseEmbedMedia(
	ctx *models.DownloadContext,
	body []byte,
) ([]*models.Media, error) {
	var mediaList []*models.Media

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed parsing HTML: %w", err)
	}

	contentID := ctx.MatchedContentID
	contentURL := ctx.MatchedContentURL

	var caption string
	doc.Find(".BodyTextContainer").Each(func(i int, c *goquery.Selection) {
		caption = c.Text()
	})

	doc.Find(".MediaContainer, .SoloMediaContainer").Each(func(i int, container *goquery.Selection) {
		container.Find("video").Each(func(j int, vid *goquery.Selection) {
			sourceEl := vid.Find("source")
			src, exists := sourceEl.Attr("src")
			if exists {
				media := ctx.Extractor.NewMedia(
					contentID,
					contentURL,
				)
				media.SetCaption(caption)
				media.AddFormat(&models.MediaFormat{
					Type:       enums.MediaTypeVideo,
					FormatID:   "video",
					URL:        []string{src},
					VideoCodec: enums.MediaCodecAVC,
					AudioCodec: enums.MediaCodecAAC,
				})
				mediaList = append(mediaList, media)
			}
		})
		container.Find("img").Each(func(j int, img *goquery.Selection) {
			src, exists := img.Attr("src")
			if exists {
				media := ctx.Extractor.NewMedia(
					contentID,
					contentURL,
				)
				media.SetCaption(caption)
				media.AddFormat(&models.MediaFormat{
					Type:     enums.MediaTypePhoto,
					FormatID: "image",
					URL:      []string{src},
				})
				mediaList = append(mediaList, media)
			}
		})
	})

	return mediaList, nil
}
