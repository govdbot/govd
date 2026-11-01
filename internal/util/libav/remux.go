package libav

import (
	"os"
	"time"

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

	err := ffmpeg.Input(inputPath, inputOptions()).
		Output(outputPath, ffmpeg.KwArgs{
			mapOption:      "0",
			"c":            copyCodec,
			movflagsOption: fastStartFlag,
		}).
		Silent(true).
		OverWriteOutput().
		WithTimeout(60 * time.Second).
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
	err := ffmpeg.Input(inputPath, inputOptions()).
		Output(tempPath, ffmpeg.KwArgs{
			mapOption:      "0",
			"c":            copyCodec,
			movflagsOption: fastStartFlag,
		}).
		Silent(true).
		OverWriteOutput().
		WithTimeout(60 * time.Second).
		Run()

	if err != nil {
		os.Remove(tempPath)
		return err
	}

	err = ffmpeg.Input(tempPath, inputOptions()).
		Output(outputPath, ffmpeg.KwArgs{
			mapOption:      "0",
			"c":            copyCodec,
			movflagsOption: fastStartFlag,
		}).
		Silent(true).
		OverWriteOutput().
		WithTimeout(60 * time.Second).
		Run()

	os.Remove(tempPath)

	if err != nil {
		os.Remove(outputPath)
		return err
	}

	return nil
}
