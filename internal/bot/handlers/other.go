package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
)

func CloseHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	settings, err := util.SettingsFromContext(ctx)
	if err != nil {
		return err
	}
	localizer := localization.New(settings.Language)
	isAdmin := util.CheckAdminPermission(bot, ctx, localizer)
	if !isAdmin {
		return nil
	}
	ctx.CallbackQuery.Answer(bot, nil)
	ctx.EffectiveMessage.Delete(bot, nil)
	return nil
}
