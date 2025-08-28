package handlers

import (
	"context"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
)

func CloseHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	chat := ctx.EffectiveChat
	res, err := database.Q().GetOrCreateChat(
		context.Background(),
		database.GetOrCreateChatParams{
			ChatID: chat.Id,
			Type:   database.ChatTypeGroup,
		},
	)
	if err != nil {
		return err
	}
	localizer := localization.New(res.Language)
	isAdmin := CheckAdminPermission(bot, ctx, localizer)
	if !isAdmin {
		return nil
	}
	ctx.CallbackQuery.Answer(bot, nil)
	ctx.EffectiveMessage.Delete(bot, nil)
	return nil
}
