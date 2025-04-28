package ninegag

import (
	"fmt"
	"strings"

	"govd/enums"
	"govd/models"

	"github.com/pkg/errors"
)

func FindBestPhoto(
	images map[string]*Media,
) (*Media, error) {
	var bestPhoto *Media
	var maxWidth int

	for _, photo := range images {
		if !strings.HasSuffix(photo.URL, ".jpg") {
			continue
		}
		if photo.Width > maxWidth {
			maxWidth = photo.Width
			bestPhoto = photo
		}
	}

	if bestPhoto == nil {
		return nil, errors.New("no photo found in post")
	}

	return bestPhoto, nil
}

func ParseVideoFormats(
	images map[string]*Media,
) ([]*models.MediaFormat, error) {
	var video *Media
	var thumbnailURL string

	for _, media := range images {
		if media.Duration > 0 {
			video = media
		}
		if strings.HasSuffix(media.URL, ".jpg") {
			thumbnailURL = media.URL
		}
	}
	if video == nil {
		return nil, errors.New("no video found in post")
	}

	codecMapping := map[string]struct {
		Field string
		Codec enums.MediaCodec
	}{
		"url":     {"URL", enums.MediaCodecAVC},
		"h265Url": {"H265URL", enums.MediaCodecHEVC},
		"vp8Url":  {"Vp8URL", enums.MediaCodecVP8},
		"vp9Url":  {"Vp9URL", enums.MediaCodecVP9},
		"av1Url":  {"Av1URL", enums.MediaCodecAV1},
	}

	formats := make([]*models.MediaFormat, 0, len(codecMapping))
	
	for _, mapping := range codecMapping {
		url := getField(video, mapping.Field)
		if url == "" {
			continue
		}

		format := &models.MediaFormat{
			FormatID:   fmt.Sprintf("video_%s", mapping.Codec),
			Type:       enums.MediaTypeVideo,
			VideoCodec: mapping.Codec,
			AudioCodec: enums.MediaCodecAAC,
			URL:        []string{url},
			Width:      int64(video.Width),
			Height:     int64(video.Height),
			Duration:   int64(video.Duration),
		}

		if thumbnailURL != "" {
			format.Thumbnail = []string{thumbnailURL}
		}

		formats = append(formats, format)
	}

	return formats, nil
}

func getField(media *Media, fieldName string) string {
	switch fieldName {
	case "URL":
		return media.URL
	case "H265URL":
		return media.H265URL
	case "Vp8URL":
		return media.Vp8URL
	case "Vp9URL":
		return media.Vp9URL
	case "Av1URL":
		return media.Av1URL
	default:
		return ""
	}
}
