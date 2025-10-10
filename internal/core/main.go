package core

import (
	"os"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

func HandleDownloadTask(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
) error {
	resp, err := extractorCtx.Extractor.GetFunc(extractorCtx)
	if err != nil {
		return err
	}
	if resp.Media == nil || len(resp.Media.Items) == 0 {
		return ext.EndGroups
	}

	formats, err := downloadMediaFormats(extractorCtx, resp.Media)
	if err != nil {
		return err
	}

	// clean up every file after
	defer func() {
		for _, fmt := range formats {
			os.Remove(fmt.FilePath)
			os.Remove(fmt.ThumbnailFilePath)
		}
	}()

	caption := formatCaption(
		resp.Media,
		extractorCtx.Settings.Captions,
	)

	_, err = SendFormats(
		bot, ctx, extractorCtx,
		resp.Media, formats,
		&models.SendFormatsOptions{
			Caption: caption,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func checkAlbumLimit(n int, settings *database.GetOrCreateChatRow) error {
	if settings.Type == database.ChatTypeGroup {
		if n > int(settings.MediaAlbumLimit) {
			return util.ErrMediaAlbumLimitExceeded
		}
	}
	return nil
}

func ValidateFormat(fmt *models.MediaFormat) error {
	if util.ExceedsMaxFileSize(fmt.FileSize) {
		return util.ErrFileTooLarge
	}
	if util.ExceedsMaxDuration(fmt.Duration) {
		return util.ErrDurationTooLong
	}
	return nil
}
