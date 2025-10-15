package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util/libav"
)

var MergeAudio = &models.Plugin{
	ID: "merge_audio",
	RunFunc: func(ctx *models.ExtractorContext, item *models.MediaItem, format *models.DownloadedFormat) error {
		filePath := format.FilePath

		audioFormat := item.GetDefaultAudioFormat()
		if audioFormat == nil {
			return fmt.Errorf("no audio format found")
		}

		if ctx.DownloadFunc == nil {
			return fmt.Errorf("download function not available in context")
		}

		downloadedAudioFormat, err := ctx.DownloadFunc(ctx, 0, audioFormat)
		if err != nil {
			return fmt.Errorf("failed to download audio format: %w", err)
		}

		outputPath := strings.TrimSuffix(
			filePath,
			filepath.Ext(filePath),
		) + "_remuxed" + filepath.Ext(filePath)

		ctx.FilesTracker.Add(&outputPath)

		err = libav.MergeVideoWithAudio(
			format.FilePath,
			downloadedAudioFormat.FilePath,
			outputPath,
		)
		if err != nil {
			return fmt.Errorf("failed to merge video with audio: %w", err)
		}

		os.Rename(outputPath, filePath)

		return nil
	},
}
