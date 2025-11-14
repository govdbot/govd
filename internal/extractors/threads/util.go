package threads

import (
	"bytes"
	"fmt"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"

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

func ParseEmbedMedia(ctx *models.ExtractorContext, body []byte) (*models.Media, error) {
	media := ctx.NewMedia()

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed parsing HTML: %w", err)
	}

	var caption string
	doc.Find(".BodyTextContainer").Each(func(i int, c *goquery.Selection) {
		caption = c.Text()
	})
	media.SetCaption(caption)

	doc.Find(".MediaContainer, .SoloMediaContainer").Each(func(i int, container *goquery.Selection) {
		container.Find("video").Each(func(j int, vid *goquery.Selection) {
			sourceEl := vid.Find("source")
			src, exists := sourceEl.Attr("src")
			if exists {
				item := media.NewItem()
				item.AddFormats(&models.MediaFormat{
					Type:       database.MediaTypeVideo,
					FormatID:   "video",
					URL:        []string{src},
					VideoCodec: database.MediaCodecAvc,
					AudioCodec: database.MediaCodecAac,
				})
			}
		})
		container.Find("img").Each(func(j int, img *goquery.Selection) {
			src, exists := img.Attr("src")
			if exists {
				item := media.NewItem()
				item.AddFormats(&models.MediaFormat{
					Type:     database.MediaTypePhoto,
					FormatID: "image",
					URL:      []string{src},
				})
			}
		})
	})

	return media, nil
}
