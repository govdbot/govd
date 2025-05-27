package plugins

import (
	"context"
	"fmt"

	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/libav"

	"github.com/pkg/errors"
)

func MergeAudio(
	media *models.DownloadedMedia,
	downloadConfig *models.DownloadConfig,
) error {
	audioFormat := media.Media.GetDefaultAudioFormat()
	if audioFormat == nil {
		return errors.New("no audio format found")
	}

	// disable remuxing
	downloadConfigCopy := *downloadConfig
	downloadConfigCopy.Remux = false

	// download the audio file
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var audioFile string
	var err error

	if len(audioFormat.Segments) == 0 {
		audioFile, err = util.DownloadFile(
			ctx, audioFormat.URL,
			audioFormat.GetFileName(),
			&downloadConfigCopy,
		)
	} else {
		audioFile, err = util.DownloadFileWithSegments(
			ctx, audioFormat.Segments,
			audioFormat.GetFileName(),
			&downloadConfigCopy,
		)
	}
	if err != nil {
		return fmt.Errorf("failed to download audio file: %w", err)
	}

	err = libav.MergeVideoWithAudio(
		media.FilePath,
		audioFile,
	)
	if err != nil {
		return fmt.Errorf("failed to merge video with audio: %w", err)
	}

	return nil
}
