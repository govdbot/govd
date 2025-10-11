package core

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

func SendFormats(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
	media *models.Media,
	formats []*models.DownloadedFormat,
	options *models.SendFormatsOptions,
) ([]gotgbot.Message, error) {
	var chatID int64
	var messageOptions *gotgbot.SendMediaGroupOpts

	chatSettings := extractorCtx.Settings

	if chatSettings.Type == database.ChatTypeGroup {
		if len(formats) > int(chatSettings.MediaAlbumLimit) {
			return nil, util.ErrMediaAlbumLimitExceeded
		}
		if !chatSettings.Nsfw && media.NSFW {
			return nil, util.ErrNSFWNotAllowed
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

	mediaGroupChunks := slices.Collect(slices.Chunk(formats, 10))

	for _, chunk := range mediaGroupChunks {
		var inputMediaList []gotgbot.InputMedia
		for i, f := range chunk {
			var caption string
			if i == 0 {
				caption = options.Caption
			}
			inputMedia, err := f.Format.GetInputMedia(
				f.FilePath, f.ThumbnailFilePath,
				caption, options.IsSpoiler,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to get input media: %w", err)
			}
			inputMediaList = append(inputMediaList, inputMedia)
		}

		util.SendMediaAction(bot, chatID, chunk[0].Format.Type)

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
			// avoid floodwait
			if len(mediaGroupChunks) > 1 {
				time.Sleep(3 * time.Second)
			}
		}
	}
	if len(sentMessages) == 0 {
		return nil, errors.New("no messages sent")
	}

	if !options.IsStored {
		err := StoreMedia(
			extractorCtx.Context,
			extractorCtx.Extractor,
			media, sentMessages,
			formats,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to cache formats: %w", err)
		}
	}
	return sentMessages, nil
}
