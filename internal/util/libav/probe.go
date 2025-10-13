package libav

import (
	"encoding/json"
	"strconv"

	"github.com/govdbot/govd/internal/logger"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

type VideoProbeData struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int32  `json:"width"`
		Height    int32  `json:"height"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func ExtractVideoMetadata(inputPath string) (int32, int32, int32) {
	logger.L.Debugf("extracting video metadata: %s", inputPath)

	data, err := ffmpeg.Probe(inputPath)
	if err != nil {
		return 0, 0, 0
	}

	probeData := &VideoProbeData{}
	err = json.Unmarshal([]byte(data), probeData)
	if err != nil {
		return 0, 0, 0
	}

	var width, height, duration int32
	for _, s := range probeData.Streams {
		if s.CodecType == "video" {
			width = s.Width
			height = s.Height
			break
		}
	}

	d, err := strconv.ParseFloat(probeData.Format.Duration, 32)
	if err == nil {
		duration = int32(d)
	}

	return width, height, duration
}
