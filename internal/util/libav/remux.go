package libav

import (
	"os"

	"github.com/govdbot/govd/internal/extractors/twitter"
	"github.com/govdbot/govd/internal/logger"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func RemuxFile(inputPath string, outputPath string) error {
	isVork := twitter.IsVorkMuxer(inputPath)
	if isVork {
		return RemuxFileWithDoublePass(inputPath, outputPath)
	}

	logger.L.Debugf("remuxing file: %s", inputPath)

	err := ffmpeg.Input(inputPath).
		Output(outputPath, ffmpeg.KwArgs{
			"map":      "0",
			"c":        "copy",
			"movflags": "+faststart",
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

func RemuxFileWithDoublePass(inputPath string, outputPath string) error {
	logger.L.Debugf("remuxing file with double pass: %s", inputPath)

	tempPath := inputPath + ".temp.mkv"
	err := ffmpeg.Input(inputPath).
		Output(tempPath, ffmpeg.KwArgs{
			"map":      "0",
			"c":        "copy",
			"movflags": "+faststart",
		}).
		Silent(true).
		OverWriteOutput().
		Run()

	if err != nil {
		os.Remove(tempPath)
		return err
	}

	err = ffmpeg.Input(tempPath).
		Output(outputPath, ffmpeg.KwArgs{
			"map":      "0",
			"c":        "copy",
			"movflags": "+faststart",
		}).
		Silent(true).
		OverWriteOutput().
		Run()

	os.Remove(tempPath)

	if err != nil {
		os.Remove(outputPath)
		return err
	}

	return nil
}
