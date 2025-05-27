package pinterest

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util/parser"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

var headers = map[string]string{
	// fix 403 error
	"X-Pinterest-Pws-Handler": "www/[username].js",
}

func ParseVideoObject(videoObj *Videos) ([]*models.MediaFormat, error) {
	var formats []*models.MediaFormat

	for key, video := range videoObj.VideoList {
		if !strings.Contains(key, "HLS") {
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
			zap.S().Debugf("extracting HLS formats: %s", key)
			hlsFormats, err := parser.ParseM3U8FromURL(video.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to extract hls formats: %w", err)
			}
			formats = make([]*models.MediaFormat, 0, len(hlsFormats))
			for _, hlsFormat := range hlsFormats {
				hlsFormat.Duration = video.Duration / 1000
				hlsFormat.Thumbnail = []string{video.Thumbnail}
				formats = append(formats, hlsFormat)
			}
		}
	}

	return formats, nil
}

func BuildPinRequestParams(pinID string) string {
	options := map[string]any{
		"options": map[string]any{
			"field_set_key": "unauth_react_main_pin",
			"id":            pinID,
		},
	}
	jsonData, _ := sonic.ConfigFastest.Marshal(options)
	params := map[string]string{
		"data": string(jsonData),
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	return values.Encode()
}
