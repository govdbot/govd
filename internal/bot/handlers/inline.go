package handlers

import (
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/core"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func InlineHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	url := strings.TrimSpace(ctx.InlineQuery.Query)
	if url == "" {
		ctx.InlineQuery.Answer(
			bot, []gotgbot.InlineQueryResult{},
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime:  1,
				IsPersonal: true,
			},
		)
		return ext.EndGroups
	}

	extractorCtx := extractors.FromURL(url)
	if extractorCtx == nil || extractorCtx.Extractor == nil {
		ctx.InlineQuery.Answer(
			bot, []gotgbot.InlineQueryResult{},
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime:  1,
				IsPersonal: true,
			},
		)
		return ext.EndGroups
	}

	settings, err := util.SettingsFromContext(ctx)
	if err != nil {
		logger.L.Errorf("failed to get settings from context: %v", err)
		extractorCtx.CancelFunc()
		return ext.EndGroups
	}
	extractorCtx.SetSettings(settings)

	err = core.HandleInlineTask(bot, ctx, extractorCtx)
	if err != nil {
		extractorCtx.CancelFunc()
		core.HandleError(bot, ctx, extractorCtx, err)
	}

	return ext.EndGroups
}

func InlineResultHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	taskID := ctx.ChosenInlineResult.ResultId

	extractorCtx, ok := core.GetTask(taskID)
	if !ok {
		logger.L.Warnf("inline task not found: %s", taskID)
		return ext.EndGroups
	}
	defer core.RemoveTask(taskID)

	// cancel the context after the task is complete
	defer extractorCtx.CancelFunc()

	err := core.HandleInlineResultTask(bot, ctx, extractorCtx)
	if err != nil {
		core.HandleError(bot, ctx, extractorCtx, err)
	}
	return ext.EndGroups
}

func InlineLoadingHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	settings, err := util.SettingsFromContext(ctx)
	if err != nil {
		return err
	}
	localizer := localization.New(settings.Language)

	ctx.CallbackQuery.Answer(bot, &gotgbot.AnswerCallbackQueryOpts{
		Text: localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.InlineLoadingMessage.ID,
		}),
		ShowAlert: true,
	})
	return nil
}
