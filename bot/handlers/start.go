package handlers

import (
	"fmt"

	"github.com/govdbot/govd/config"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

var startMessage = "govd is an open-source telegram bot " +
	"that allows you to download medias from " +
	"various platforms.\n\n" +
	"learn how to use this bot by clicking the " +
	"'usage' button below. you can find the list of " +
	"supported platforms with the 'extractors' button."

func getStartKeyboard(bot *gotgbot.Bot) gotgbot.InlineKeyboardMarkup {
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{
					Text: "add to group",
					Url: fmt.Sprintf(
						"https://t.me/%s?startgroup=true",
						bot.Username,
					),
				},
			},
			{
				{
					Text:         "usage",
					CallbackData: "help",
				},
				{
					Text:         "stats",
					CallbackData: "stats",
				},
			},
			{
				{
					Text:         "extractors",
					CallbackData: "extractors",
				},
				{
					Text: "support",
					Url:  "https://t.me/govdsupport",
				},
			},
			{
				{
					Text:         "instances",
					CallbackData: "instances",
				},
				{
					Text: "github",
					Url:  config.Env.RepoURL,
				},
			},
		},
	}
}

func StartHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveMessage.Chat.Type != gotgbot.ChatTypePrivate {
		ctx.EffectiveMessage.Reply(
			bot,
			"i'm online! i'll download every media in this group.",
			nil,
		)
		return nil
	}
	keyboard := getStartKeyboard(bot)
	if ctx.Update.Message != nil {
		ctx.EffectiveMessage.Reply(
			bot,
			startMessage,
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &keyboard,
			},
		)
	} else if ctx.Update.CallbackQuery != nil {
		ctx.CallbackQuery.Answer(bot, nil)
		ctx.EffectiveMessage.EditText(
			bot,
			startMessage,
			&gotgbot.EditMessageTextOpts{
				ReplyMarkup: keyboard,
			},
		)

	}
	return nil
}
