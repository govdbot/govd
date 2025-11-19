package core

import (
	"fmt"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/google/uuid"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var tasks = expirable.NewLRU[string, *models.ExtractorContext](0, nil, 5*time.Minute)

func HandleInlineTask(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
) error {
	taskID := uuid.NewString()[:8]
	ok := AddTask(taskID, extractorCtx)
	if !ok {
		return fmt.Errorf("failed to add inline task to cache")
	}

	localizer := localization.New(extractorCtx.Chat.Language)

	inlineResult := &gotgbot.InlineQueryResultArticle{
		Id: taskID,
		Title: localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.InlineShareMessage.ID,
		}),
		InputMessageContent: &gotgbot.InputTextMessageContent{
			MessageText: localizer.T(&i18n.LocalizeConfig{
				MessageID: localization.InlineProcessingMessage.ID,
			}),
			ParseMode: gotgbot.ParseModeHTML,
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled: true,
			},
		},
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text:         "...",
						CallbackData: "inline:loading",
					},
				},
			},
		},
	}
	ok, err := ctx.InlineQuery.Answer(
		bot, []gotgbot.InlineQueryResult{inlineResult},
		&gotgbot.AnswerInlineQueryOpts{
			CacheTime:  util.Ptr(int64(0)),
			IsPersonal: true,
		},
	)
	if err != nil || !ok {
		return err
	}

	return nil
}

func HandleInlineResultTask(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
) error {
	defer extractorCtx.FilesTracker.Cleanup()

	key := extractorCtx.Key()

	taskQueue.Acquire(key)
	defer taskQueue.Release(key)

	taskResult, err := executeDownload(extractorCtx, true)
	if err != nil {
		return err
	}

	caption := formatCaption(
		taskResult.Media,
		bot.Username,
		extractorCtx.Chat.Captions,
		extractorCtx,
	)

	err = SendInlineFormats(
		bot, ctx, extractorCtx,
		taskResult.Media, taskResult.Formats,
		&models.SendFormatsOptions{
			Caption:  caption,
			IsStored: taskResult.IsStored,
		},
	)
	if err != nil {
		return err
	}

	return nil
}

func AddTask(taskID string, ctx *models.ExtractorContext) bool {
	return !tasks.Add(taskID, ctx)
}

func GetTask(taskID string) (*models.ExtractorContext, bool) {
	return tasks.Get(taskID)
}

func RemoveTask(taskID string) bool {
	return tasks.Remove(taskID)
}
