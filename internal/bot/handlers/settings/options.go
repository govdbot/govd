package settings

import (
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func SettingsOptionsHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	// settings.options.id
	parts := strings.Split(ctx.CallbackQuery.Data, ".")
	if len(parts) < 3 {
		return nil
	}
	settingID := parts[2]

	return ListOptionsByID(b, ctx, settingID)
}

func ListOptionsByID(b *gotgbot.Bot, ctx *ext.Context, settingID string) error {
	setting := GetSettingByID(settingID)
	if setting == nil {
		return nil
	}

	chat := ctx.EffectiveChat
	isGroup := chat.Type != gotgbot.ChatTypePrivate

	settings, err := util.SettingsFromContext(ctx)
	if err != nil {
		return err
	}

	localizer := localization.New(settings.Language)
	if isGroup && !util.CheckAdminPermission(b, ctx, localizer) {
		return nil
	}

	ctx.CallbackQuery.Answer(b, nil)
	text := localizer.T(&i18n.LocalizeConfig{
		MessageID: setting.DescriptionKey,
	})

	buttons := BuildSettingsOptionsButtons(setting, settings, localizer)
	buttons = append(buttons, []gotgbot.InlineKeyboardButton{
		{
			Text: localizer.T(&i18n.LocalizeConfig{
				MessageID: localization.BackButton.ID}),
			CallbackData: "settings",
		},
	})

	ctx.EffectiveMessage.EditText(
		b,
		text,
		&gotgbot.EditMessageTextOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			},
		},
	)

	return nil
}
