package pinterest

import (
	"fmt"
	"govd/enums"
	"govd/models"
	"govd/util/parser"

	"github.com/bytedance/sonic"
)

func ParseVideoObject(videoObj *Videos) ([]*models.MediaFormat, error) {
	var formats []*models.MediaFormat

	for key, video := range videoObj.VideoList {
		if key != "HLS" {
			formats = append(formats, &models.MediaFormat{
				FormatID:   key,
				URL:        []string{video.URL},
				Type:       enums.MediaTypeVideo,
				VideoCodec: enums.MediaCodecAVC,
				AudioCodec: enums.MediaCodecAAC,
				Width:      video.Width,
				Height:     video.Height,
				Duration:   video.Duration / 1000,
				Thumbnail:  []string{video.Thumbnail},
			})
		} else {
			hlsFormats, err := parser.ParseM3U8FromURL(video.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to extract hls formats: %w", err)
			}
			for _, hlsFormat := range hlsFormats {
				hlsFormat.Duration = video.Duration / 1000
				hlsFormat.Thumbnail = []string{video.Thumbnail}
				formats = append(formats, hlsFormat)
			}
		}
	}

	return formats, nil
}

func BuildPinRequestParams(pinID string) map[string]string {
	options := map[string]interface{}{
		"options": map[string]interface{}{
			"field_set_key": "unauth_react_main_pin",
			"id":            pinID,
		},
	}

	jsonData, _ := sonic.ConfigFastest.Marshal(options)
	return map[string]string{
		"data": string(jsonData),
	}
}
