package libav

import (
	"fmt"
	"os"
	"time"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func MergeVideoWithAudio(
	videoPath string,
	audioPath string,
	outputPath string,
) error {
	err := ffmpeg.Output(
		[]*ffmpeg.Stream{
			ffmpeg.Input(videoPath, inputOptions()),
			ffmpeg.Input(audioPath, inputOptions()),
		},
		outputPath,
		ffmpeg.KwArgs{
			movflagsOption: fastStartFlag,
			"c:v":          copyCodec,
			"c:a":          copyCodec,
		}).
		Silent(true).
		OverWriteOutput().
		WithTimeout(60 * time.Second).
		Run()

	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to merge files: %w", err)
	}

	return nil
}
