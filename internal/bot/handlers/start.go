package handlers

import (
	"fmt"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func StartHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	if ctx.EffectiveChat.Type != gotgbot.ChatTypePrivate {
		return HandleGroupStart(bot, ctx)
	}
	user := ctx.EffectiveUser

	chat, err := util.ChatFromContext(ctx)
	if err != nil {
		return err
	}
	localizer := localization.New(chat.Language)

	keyboard := getStartKeyboard(bot, localizer)

	text := localizer.T(&i18n.LocalizeConfig{
		MessageID: localization.StartMessage.ID,
		TemplateData: map[string]string{
			"Name": util.MentionUser(user),
		},
	})

	if ctx.Message != nil {
		ctx.EffectiveMessage.Reply(
			bot, text,
			&gotgbot.SendMessageOpts{
				ReplyMarkup: keyboard,
			},
		)
	} else if ctx.CallbackQuery != nil {
		ctx.CallbackQuery.Answer(bot, nil)
		ctx.EffectiveMessage.EditText(
			bot, text,
			&gotgbot.EditMessageTextOpts{
				ReplyMarkup: keyboard,
			},
		)
	}
	return nil
}

func getStartKeyboard(
	bot *gotgbot.Bot,
	localizer *localization.Localizer,
) gotgbot.InlineKeyboardMarkup {
	addButton := localizer.T(&i18n.LocalizeConfig{
		MessageID: localization.AddButton.ID,
	})
	settingsButton := localizer.T(&i18n.LocalizeConfig{
		MessageID: localization.SettingsButton.ID,
	})
	return gotgbot.InlineKeyboardMarkup{
		InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
			{
				{
					Text: addButton,
					Url: fmt.Sprintf(
						"https://t.me/%s?startgroup=true",
						bot.Username,
					),
				},
			},
			{
				{
					Text:         settingsButton,
					CallbackData: "settings",
				},
			},
			{
				{
					Text: "github",
					Url:  config.Env.RepoURL,
				},
			},
		},
	}
}

func HandleGroupStart(bot *gotgbot.Bot, ctx *ext.Context) error {
	ctx.EffectiveMessage.Reply(
		bot,
		"✅",
		nil,
	)
	return nil
}
