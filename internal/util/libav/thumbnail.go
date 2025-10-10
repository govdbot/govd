package libav

import (
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func ExtractVideoThumbnail(
	videoPath string,
	outputPath string,
) (string, error) {
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
