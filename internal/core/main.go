package core

import (
	"fmt"
	"os"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"golang.org/x/sync/singleflight"
)

var sf singleflight.Group

func HandleDownloadTask(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
) error {
	message := ctx.EffectiveMessage
	isSpoiler := util.HasHashtagEntity(message, "spoiler") ||
		util.HasHashtagEntity(message, "nsfw")

	// use singleflight to deduplicate
	// parallel downloads of the same content
	key := extractorCtx.Extractor.ID + ":" + extractorCtx.ContentID
	result, err, shared := sf.Do(key, func() (interface{}, error) {
		return executeDownload(extractorCtx)
	})
	if err != nil {
		return err
	}
	if shared {
		logger.L.Debugf("shared download result for key: %s", key)
	}

	taskResult := result.(*models.TaskResult)

	err = checkAlbumLimit(len(taskResult.Media.Items), extractorCtx.Settings)
	if err != nil {
		return err
	}

	// clean up every file after task completes
	defer func() {
		if taskResult.IsStored {
			return
		}
		for _, fmt := range taskResult.Formats {
			os.Remove(fmt.FilePath)
			os.Remove(fmt.ThumbnailFilePath)
		}
	}()

	caption := formatCaption(
		taskResult.Media,
		extractorCtx.Settings.Captions,
	)

	_, err = SendFormats(
		bot, ctx, extractorCtx,
		taskResult.Media, taskResult.Formats,
		&models.SendFormatsOptions{
			Caption:   caption,
			IsSpoiler: isSpoiler,
			IsStored:  taskResult.IsStored,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// performs the actual download operation
// this function is wrapped by singleflight
// to prevent duplicate downloads
func executeDownload(extractorCtx *models.ExtractorContext) (*models.TaskResult, error) {
	storedMedia, err := database.Q().GetMediaByContentID(
		extractorCtx.Context,
		database.GetMediaByContentIDParams{
			ExtractorID: extractorCtx.Extractor.ID,
			ContentID:   extractorCtx.ContentID,
		},
	)
	if err == nil {
		logger.L.Debugf(
			"found stored media: %s/%s",
			extractorCtx.Extractor.ID,
			extractorCtx.ContentID,
		)
		media, err := ParseStoredMedia(
			extractorCtx.Context,
			extractorCtx.Extractor,
			&storedMedia,
		)
		if err != nil {
			return nil, err
		}
		formats := make([]*models.DownloadedFormat, 0, len(media.Items))
		for i, item := range media.Items {
			formats = append(formats, &models.DownloadedFormat{
				Format: item.Formats[0],
				Index:  i,
			})
		}
		return &models.TaskResult{
			Media:    media,
			Formats:  formats,
			IsStored: true,
		}, nil
	}

	resp, err := extractorCtx.Extractor.GetFunc(extractorCtx)
	if err != nil {
		return nil, err
	}
	if resp.Media == nil || len(resp.Media.Items) == 0 {
		return nil, fmt.Errorf("no media found")
	}

	formats, err := downloadMediaFormats(extractorCtx, resp.Media)
	if err != nil {
		return nil, err
	}

	return &models.TaskResult{
		Media:   resp.Media,
		Formats: formats,
	}, nil
}

func checkAlbumLimit(n int, settings *database.GetOrCreateChatRow) error {
	if settings.Type == database.ChatTypeGroup {
		if n > int(settings.MediaAlbumLimit) {
			return util.ErrMediaAlbumLimitExceeded
		}
	}
	// global limit
	// TODO: make this configurable
	if n > 30 {
		return util.ErrMediaAlbumGlobalLimitExceeded
	}
	return nil
}

func validateFormat(fmt *models.MediaFormat) error {
	if util.ExceedsMaxFileSize(fmt.FileSize) {
		return util.ErrFileTooLarge
	}
	if util.ExceedsMaxDuration(fmt.Duration) {
		return util.ErrDurationTooLong
	}
	return nil
}
