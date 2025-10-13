package libav

import (
	"os"

	"github.com/govdbot/govd/internal/logger"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func ExtractVideoThumbnail(
	videoPath string,
	outputPath string,
) (string, error) {
	logger.L.Debugf("extracting thumbnail from video: %s", videoPath)

	err := ffmpeg.Input(videoPath).
		Filter("select", ffmpeg.Args{"gte(n,0)"}).
		Output(outputPath, ffmpeg.KwArgs{
			"vframes": 1,
			"vcodec":  "mjpeg",
		}).
		Silent(true).
		OverWriteOutput().
		Run()

	if err != nil {
		os.Remove(outputPath)
		return "", err
	}

	return outputPath, nil
}
