package ninegag

import (
	"fmt"
	"strings"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
)

func FindBestPhoto(images map[string]*Media) (*Media, error) {
	var bestPhoto *Media
	var maxWidth int32

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
		return nil, fmt.Errorf("no suitable photo found")
	}

	return bestPhoto, nil
}

func ParseVideoFormats(images map[string]*Media) ([]*models.MediaFormat, error) {
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
		return nil, fmt.Errorf("no video found")
	}

	codecMapping := map[string]struct {
		Field string
		Codec database.MediaCodec
	}{
		"url":     {"URL", database.MediaCodecAvc},
		"h265Url": {"H265URL", database.MediaCodecHevc},
		"vp8Url":  {"Vp8URL", database.MediaCodecVp8},
		"vp9Url":  {"Vp9URL", database.MediaCodecVp9},
		"av1Url":  {"Av1URL", database.MediaCodecAv1},
	}

	formats := make([]*models.MediaFormat, 0, len(codecMapping))

	for _, mapping := range codecMapping {
		url := getField(video, mapping.Field)
		if url == "" {
			continue
		}

		format := &models.MediaFormat{
			FormatID:   "video_" + string(mapping.Codec),
			Type:       database.MediaTypeVideo,
			VideoCodec: mapping.Codec,
			AudioCodec: database.MediaCodecAac,
			URL:        []string{url},
			Width:      video.Width,
			Height:     video.Height,
			Duration:   video.Duration,
		}
		if thumbnailURL != "" {
			format.ThumbnailURL = []string{thumbnailURL}
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
