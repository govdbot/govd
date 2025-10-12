package libav

import (
	"github.com/govdbot/govd/internal/logger"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func RemuxVideo(inputPath string, outputPath string) error {
	logger.L.Debugf("remuxing video %s to %s", inputPath, outputPath)

	return ffmpeg.Input(inputPath).
		Output(outputPath, ffmpeg.KwArgs{
			"c:v": "copy",
			"c:a": "copy",
		}).
		Silent(true).
		OverWriteOutput().
		Run()
}
