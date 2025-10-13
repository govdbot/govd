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
	logger.L.Errorf("unexpected error: %v", err)

	currentErr := err
	for currentErr != nil {
		var botError *util.Error
		if errors.As(currentErr, &botError) {
			HandleBotError(b, ctx, extractorCtx, botError)
			return
		}
		currentErr = errors.Unwrap(currentErr)
	}

	localizer := localization.New(extractorCtx.Settings.Language)

	errorString := err.Error()
	errorID := util.HashedError(errorString)

	errorMessage := "⚠️ " + localizer.T(&i18n.LocalizeConfig{
		MessageID: localization.ErrorMessage.ID,
	}) + " [<code>" + errorID + "</code>]"

	if !extractorCtx.Settings.Silent {
		ctx.EffectiveMessage.Reply(b, errorMessage, nil)
	}

	database.Q().LogError(
		extractorCtx.Context,
		database.LogErrorParams{
			ID:      errorID,
			Message: errorString,
		},
	)

}

func HandleBotError(
	b *gotgbot.Bot,
	ctx *ext.Context,
	extractorCtx *models.ExtractorContext,
	err *util.Error,
) {
	if extractorCtx.Settings.Silent {
		return
	}

	localizer := localization.New(extractorCtx.Settings.Language)
	errorMessage := "⚠️ " + localizer.T(&i18n.LocalizeConfig{
		MessageID: err.ID,
	})

	ctx.EffectiveMessage.Reply(b, errorMessage, nil)
}
