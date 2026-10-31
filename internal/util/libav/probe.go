package libav

import (
	"encoding/json"
	"strconv"
	"time"

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

	options := inputOptions()
	options["show_entries"] = "stream=codec_type,width,height:format=duration"
	data, err := ffmpeg.ProbeWithTimeout(inputPath, 15*time.Second, options)
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
