package core

import (
	"errors"
	"strings"

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
	chat := extractorCtx.Chat
	localizer := localization.New(chat.Language)

	botError := asBotError(err)
	if botError != nil {
		sendErrorMessage(
			b, ctx, "",
			localizer.T(&i18n.LocalizeConfig{
				MessageID: botError.ID,
			}),
		)
		return
	}

	if errors.Is(err, ErrNoMedia) {
		return
	}
	if isChatWriteForbidden(err) {
		return
	}
	if isPermissionDenied(err) {
		sendErrorMessage(
			b, ctx, "",
			localizer.T(&i18n.LocalizeConfig{
				MessageID: localization.ErrorPermissionDenied.ID,
			}),
		)
		return
	}

	errorID := util.HashedError(err)

	logger.L.Errorf("unexpected error: [%s] %v", errorID, err)

	sendErrorMessage(
		b, ctx, errorID,
		localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.ErrorMessage.ID,
		}),
	)

	database.Q().LogError(
		extractorCtx.Context,
		database.LogErrorParams{
			ID:      errorID,
			Message: err.Error(),
		},
	)
}

func isChatWriteForbidden(err error) bool {
	return strings.Contains(err.Error(), "CHAT_WRITE_FORBIDDEN")
}

func isPermissionDenied(err error) bool {
	return strings.Contains(err.Error(), "not enough rights")
}

func asBotError(err error) *util.Error {
	currentErr := err
	for currentErr != nil {
		var botError *util.Error
		if errors.As(currentErr, &botError) {
			return botError
		}
		currentErr = errors.Unwrap(currentErr)
	}
	return nil
}

func formatErrorMessage(ctx *ext.Context, message string, errorID string) string {
	var suffix string
	if errorID != "" {
		if ctx.CallbackQuery != nil || ctx.InlineQuery != nil {
			suffix = " [" + errorID + "]"
		} else {
			suffix = " [<code>" + errorID + "</code>]"
		}
	}
	return "⚠️ " + message + suffix
}

func sendErrorMessage(
	b *gotgbot.Bot,
	ctx *ext.Context,
	errroID string,
	message string,
) {
	message = formatErrorMessage(ctx, message, errroID)

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
