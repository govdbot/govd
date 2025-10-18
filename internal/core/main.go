package core

import (
	"fmt"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
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
	defer extractorCtx.FilesTracker.Cleanup()

	message := ctx.EffectiveMessage
	isSpoiler := util.HasHashtagEntity(message, "spoiler") ||
		util.HasHashtagEntity(message, "nsfw")

	taskResult, err := getDownloadResult(extractorCtx, false)
	if err != nil {
		return err
	}

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
func executeDownload(extractorCtx *models.ExtractorContext, isInline bool) (*models.TaskResult, error) {
	if config.Env.Caching {
		task, err := taskFromDatabase(extractorCtx)
		if err == nil {
			if isInline && len(task.Media.Items) > 1 {
				return nil, util.ErrInlineMediaAlbum
			}
			err = checkAlbumLimit(len(task.Media.Items), extractorCtx.Settings)
			if err != nil {
				return nil, err
			}
			logger.L.Debugf(
				"media found in database: %s/%s",
				extractorCtx.Extractor.ID,
				extractorCtx.ContentID,
			)
			return task, nil
		}
	}
	resp, err := extractorCtx.Extractor.GetFunc(extractorCtx)
	if err != nil {
		return nil, err
	}
	if resp.Media == nil || len(resp.Media.Items) == 0 {
		return nil, fmt.Errorf("no media found")
	}

	if isInline && len(resp.Media.Items) > 1 {
		return nil, util.ErrInlineMediaAlbum
	}
	err = checkAlbumLimit(len(resp.Media.Items), extractorCtx.Settings)
	if err != nil {
		return nil, err
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

func taskFromDatabase(ctx *models.ExtractorContext) (*models.TaskResult, error) {
	mediaRow, err := database.Q().GetMediaByContentID(
		ctx.Context,
		database.GetMediaByContentIDParams{
			ExtractorID: ctx.Extractor.ID,
			ContentID:   ctx.ContentID,
		},
	)
	if err != nil {
		return nil, err
	}

	media, err := ParseStoredMedia(ctx.Context, ctx.Extractor, &mediaRow)
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

func getDownloadResult(ctx *models.ExtractorContext, isInline bool) (*models.TaskResult, error) {
	key := ctx.Extractor.ID + "/" + ctx.ContentID
	result, err, shared := sf.Do(key, func() (interface{}, error) {
		return executeDownload(ctx, isInline)
	})
	if err != nil {
		return nil, err
	}
	if shared {
		logger.L.Debugf("shared download result for key: %s", key)
	}
	return result.(*models.TaskResult), nil
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
