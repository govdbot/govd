package handlers

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func AddedToGroupHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if !isAddedUpdate(bot, ctx) {
		return ext.EndGroups
	}
	chat, err := util.ChatFromContext(ctx)
	if err != nil {
		return err
	}
	localizer := localization.New(chat.Language)
	bot.SendMessage(
		chat.ChatID,
		localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.AddedToGroupMessage.ID,
		}),
		nil,
	)
	return ext.EndGroups
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
