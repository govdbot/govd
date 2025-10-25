package core

import (
	"errors"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func HandleError(
	b *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
	err error,
) {
	currentErr := err
	for currentErr != nil {
		var botError *util.Error
		if errors.As(currentErr, &botError) {
			if extractorCtx.Settings.Silent {
				return
			}
			localizer := localization.New(extractorCtx.Settings.Language)
			errorMessage := "⚠️ " + localizer.T(&i18n.LocalizeConfig{
				MessageID: botError.ID,
			})
			sendErrorMessage(b, ctx, errorMessage)
			return
		}
		currentErr = errors.Unwrap(currentErr)
	}

	if errors.Is(err, NoMedia) {
		return
	}

	logger.L.Errorf("unexpected error: %v", err)

	localizer := localization.New(extractorCtx.Settings.Language)
	errorString := err.Error()
	errorID := util.HashedError(errorString)

	errorMessage := "⚠️ " + localizer.T(&i18n.LocalizeConfig{
		MessageID: localization.ErrorMessage.ID,
	})
	if ctx.CallbackQuery != nil || ctx.InlineQuery != nil {
		errorMessage += " [" + errorID + "]"
	} else {
		errorMessage += " [<code>" + errorID + "</code>]"
	}

	sendErrorMessage(b, ctx, errorMessage)

	database.Q().LogError(
		extractorCtx.Context,
		database.LogErrorParams{
			ID:      errorID,
			Message: errorString,
		},
	)

}

func sendErrorMessage(
	b *gotgbot.Bot,
	ctx *ext.Context,
	message string,
) {
	switch {
	case ctx.Message != nil:
		ctx.EffectiveMessage.Reply(b, message, nil)
	case ctx.CallbackQuery != nil:
		ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
			Text:      message,
			ShowAlert: true,
		})
	case ctx.InlineQuery != nil:
		ctx.InlineQuery.Answer(b, nil,
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime: 1,
				Button: &gotgbot.InlineQueryResultsButton{
					Text:           message,
					StartParameter: "start",
				},
			},
		)
	case ctx.ChosenInlineResult != nil:
		b.EditMessageText(
			message,
			&gotgbot.EditMessageTextOpts{
				InlineMessageId: ctx.ChosenInlineResult.InlineMessageId,
				LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
					IsDisabled: true,
				},
			},
		)
	}
}
