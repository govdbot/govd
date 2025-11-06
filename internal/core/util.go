package core

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/plugins"
	"github.com/govdbot/govd/internal/util"
	"github.com/govdbot/govd/internal/util/download"
	"github.com/govdbot/govd/internal/util/libav"
)

var ErrNoMedia = errors.New("no media found")

func parseFormatFromDB(row *database.GetMediaFormatRow) *models.MediaFormat {
	return &models.MediaFormat{
		FormatID:   row.FormatID,
		FileID:     row.FileID,
		Type:       row.Type,
		AudioCodec: row.AudioCodec.MediaCodec,
		VideoCodec: row.VideoCodec.MediaCodec,
		FileSize:   row.FileSize.Int64,
		Duration:   row.Duration.Int32,
		Title:      row.Title.String,
		Artist:     row.Artist.String,
		Width:      row.Width.Int32,
		Height:     row.Height.Int32,
		Bitrate:    row.Bitrate.Int64,
	}
}

func getThumbnail(
	ctx *models.ExtractorContext,
	format *models.MediaFormat,
	filePath string,
) (string, error) {
	fileDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(fileName)
	fileBaseName := fileName[:len(fileName)-len(fileExt)]
	thumbnailFilePath := filepath.Join(fileDir, fileBaseName+".jpeg")

	if len(format.ThumbnailURL) > 0 {
		file, err := download.DownloadFileInMemory(
			ctx, format.ThumbnailURL,
			format.DownloadSettings,
		)
		if err != nil {
			return "", err
		}
		if file == nil {
			return "", fmt.Errorf("downloaded file is nil")
		}

		var size int
		if format.Type == database.MediaTypeAudio {
			// for audio, use a smaller thumbnail
			size = 320
		}
		if err := util.ImgToJPEG(file, thumbnailFilePath, size); err != nil {
			return "", err
		}
	} else if format.Type == database.MediaTypeVideo {
		return libav.ExtractVideoThumbnail(filePath, thumbnailFilePath)
	}

	return thumbnailFilePath, nil
}

func insertVideoInfo(format *models.MediaFormat, filePath string) {
	duration, width, height := util.ExtractMP4Metadata(filePath)
	if duration == 0 && width == 0 && height == 0 {
		duration, width, height = libav.ExtractVideoMetadata(filePath)
	}
	format.Duration = duration
	format.Width = width
	format.Height = height
}

func formatCaption(media *models.Media, isEnabled bool) string {
	caption := media.Caption

	var description string
	header := strings.ReplaceAll(
		config.Env.CaptionsHeader,
		"{{url}}", media.ContentURL,
	)
	if isEnabled && caption != "" {
		if len(caption) > 600 {
			caption = caption[:600] + "..."
		}
		description = strings.ReplaceAll(
			config.Env.CaptionsDescription,
			"{{text}}", util.Unquote(caption),
		)
	}
	return header + "\n" + description
}

// utility function to merge audio into video formats with no audio
func mergeFormats(item *models.MediaItem, format *models.DownloadedFormat) {
	if format.Format.Type != database.MediaTypeVideo {
		return
	}
	if format.Format.AudioCodec != "" {
		return
	}
	audioFormat := item.GetDefaultAudioFormat()
	if audioFormat == nil {
		return
	}
	format.Format.AudioCodec = audioFormat.AudioCodec
	format.Format.Plugins = append(
		format.Format.Plugins,
		plugins.MergeAudio,
	)
}
