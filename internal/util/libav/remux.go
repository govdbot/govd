package libav

import (
	"os"

	"github.com/govdbot/govd/internal/logger"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func RemuxFile(inputPath string, outputPath string) error {
	logger.L.Debugf("remuxing file: %s", inputPath)

	err := ffmpeg.Input(inputPath).
		Output(outputPath, ffmpeg.KwArgs{
			"c:v": "copy",
			"c:a": "copy",
		}).
		Silent(true).
		OverWriteOutput().
		Run()

	if err != nil {
		os.Remove(outputPath)
		return err
	}
	return nil
}
