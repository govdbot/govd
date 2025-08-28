package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func CheckAdminPermission(b *gotgbot.Bot, ctx *ext.Context, localizer *localization.Localizer) bool {
	if !util.IsUserAdmin(b, ctx.EffectiveUser, ctx.EffectiveChat.Id) {
		noPermissionMessage := localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.NoPermission.ID,
		})

		if ctx.CallbackQuery != nil {
			ctx.CallbackQuery.Answer(b, &gotgbot.AnswerCallbackQueryOpts{
				Text:      noPermissionMessage,
				ShowAlert: true,
			})
		} else if ctx.Message != nil {
			ctx.EffectiveMessage.Reply(b, noPermissionMessage, nil)
		}
		return false
	}
	return true
}
