package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/admission"
)

func AdmissionHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveUser != nil && admission.Default.AllowLight(ctx.EffectiveUser.Id) {
		return ext.ContinueGroups
	}
	if ctx.CallbackQuery != nil {
		ctx.CallbackQuery.Answer(bot, nil)
	} else if ctx.InlineQuery != nil {
		ctx.InlineQuery.Answer(bot, []gotgbot.InlineQueryResult{}, nil)
	}
	return ext.EndGroups
}
