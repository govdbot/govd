package handlers

import (
	"slices"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
)

func WhitelistHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	var effectiveID int64
	switch {
	case ctx.EffectiveChat != nil:
		effectiveID = ctx.EffectiveChat.Id
	case ctx.EffectiveUser != nil:
		effectiveID = ctx.EffectiveUser.Id
	default:
		return ext.ContinueGroups
	}
	if !slices.Contains(config.Env.Whitelist, effectiveID) {
		if ctx.CallbackQuery != nil {
			ctx.CallbackQuery.Answer(bot, nil)
		} else if ctx.InlineQuery != nil {
			ctx.InlineQuery.Answer(bot, []gotgbot.InlineQueryResult{}, nil)
		}
		return ext.EndGroups
	}
	return ext.ContinueGroups
}
