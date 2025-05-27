package core

import (
	"context"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/pkg/errors"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func HandleDownloadRequest(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	taskCtx context.Context,
	dlCtx *models.DownloadContext,
) error {
	chatID := ctx.EffectiveMessage.Chat.Id
	if dlCtx.Extractor.Type == enums.ExtractorTypeSingle {
		TypingEffect(bot, chatID)
		err := HandleDefaultFormatDownload(bot, ctx, taskCtx, dlCtx)
		if err != nil {
			return err
		}
		return nil
	}
	return util.ErrUnsupportedExtractorType
}

func SendMedias(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	dlCtx *models.DownloadContext,
	medias []*models.DownloadedMedia,
	options *models.SendMediaFormatsOptions,
) ([]gotgbot.Message, error) {
	var chatID int64
	var messageOptions *gotgbot.SendMediaGroupOpts

	if dlCtx.GroupSettings != nil {
		if len(medias) > dlCtx.GroupSettings.MediaGroupLimit {
			return nil, util.ErrMediaGroupLimitExceeded
		}
		if !*dlCtx.GroupSettings.NSFW {
			for _, media := range medias {
				if media.Media.NSFW {
					return nil, util.ErrNSFWNotAllowed
				}
			}
		}
	}

	switch {
	case ctx.Message != nil:
		chatID = ctx.EffectiveMessage.Chat.Id
		messageOptions = &gotgbot.SendMediaGroupOpts{
			ReplyParameters: &gotgbot.ReplyParameters{
				MessageId: ctx.EffectiveMessage.MessageId,
			},
		}
	case ctx.CallbackQuery != nil:
		chatID = ctx.CallbackQuery.Message.GetChat().Id
		messageOptions = nil
	case ctx.InlineQuery != nil:
		chatID = ctx.InlineQuery.From.Id
		messageOptions = nil
	case ctx.ChosenInlineResult != nil:
		chatID = ctx.ChosenInlineResult.From.Id
		messageOptions = &gotgbot.SendMediaGroupOpts{
			DisableNotification: true,
		}
	default:
		return nil, errors.New("failed to get chat id")
	}

	var sentMessages []gotgbot.Message

	mediaGroupChunks := slices.Collect(
		slices.Chunk(medias, 10),
	)

	for _, chunk := range mediaGroupChunks {
		var inputMediaList []gotgbot.InputMedia
		for idx, media := range chunk {
			// always clean up files, in case of error
			defer func() {
				if media.FilePath != "" {
					os.Remove(media.FilePath)
				}
				if media.ThumbnailFilePath != "" {
					os.Remove(media.ThumbnailFilePath)
				}
			}()
			var caption string

			if idx == 0 {
				caption = options.Caption
			}
			inputMedia, err := media.Media.Format.GetInputMedia(
				media.FilePath,
				media.ThumbnailFilePath,
				caption,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get input media: %w", err)
			}
			inputMediaList = append(inputMediaList, inputMedia)
		}
		mediaType := chunk[0].Media.Format.Type
		SendingEffect(bot, chatID, mediaType)
		msgs, err := bot.SendMediaGroup(
			chatID,
			inputMediaList,
			messageOptions,
		)
		if err != nil {
			return nil, err
		}

		sentMessages = append(sentMessages, msgs...)
		if sentMessages[0].Chat.Type != gotgbot.ChatTypePrivate {
			if len(mediaGroupChunks) > 1 {
				time.Sleep(3 * time.Second)
			} // avoid floodwait?
		}
	}
	if len(sentMessages) == 0 {
		return nil, errors.New("no messages sent")
	}
	if !options.IsStored {
		err := StoreMedias(
			dlCtx,
			sentMessages,
			medias,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to cache formats: %w", err)
		}
	}
	return sentMessages, nil
}
