package libav

import (
	"fmt"
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func MergeVideoWithAudio(
	videoPath string,
	audioPath string,
	outputPath string,
) error {
	err := ffmpeg.Output(
		[]*ffmpeg.Stream{
			ffmpeg.Input(videoPath),
			ffmpeg.Input(audioPath),
		},
		outputPath,
		ffmpeg.KwArgs{
			"movflags": "+faststart",
			"c:v":      "copy",
			"c:a":      "copy",
		}).
		Silent(true).
		OverWriteOutput().
		Run()

	if err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("failed to merge files: %w", err)
	}

	return nil
}
