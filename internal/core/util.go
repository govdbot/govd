package core

import (
	"path/filepath"
	"strings"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/download"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/govdbot/govd/internal/util/libav"
)

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
			ctx.Context,
			ctx.HTTPClient,
			format.ThumbnailURL,
			format.DownloadSettings,
		)
		if err != nil {
			return "", err
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
