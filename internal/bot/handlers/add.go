package handlers

import (
	"context"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func AddedToGroupHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !isAddedUpdate(bot, ctx) {
		return nil
	}
	addedBy := ctx.MyChatMember.From
	chat := ctx.MyChatMember.Chat
	res, err := database.Q().GetOrCreateChat(
		context.Background(),
		database.GetOrCreateChatParams{
			ChatID:   chat.Id,
			Type:     database.ChatTypeGroup,
			Language: localization.GetLocaleFromCode(addedBy.LanguageCode),
		},
	)
	if err != nil {
		return err
	}
	localizer := localization.New(res.Language)
	chat.SendMessage(
		bot,
		localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.AddedToGroupMessage.ID,
		}),
		nil,
	)
	return nil
}

func isAddedUpdate(bot *gotgbot.Bot, ctx *ext.Context) bool {
	update := ctx.MyChatMember
	if update == nil {
		return false
	}
	if update.NewChatMember == nil {
		return false
	}
	oldStatus := update.OldChatMember.GetStatus()
	newStatus := update.NewChatMember.GetStatus()
	if (oldStatus == "kicked" || oldStatus == "left") &&
		(newStatus == "administrator" || newStatus == "member") {
		return true
	}
	return false
}
