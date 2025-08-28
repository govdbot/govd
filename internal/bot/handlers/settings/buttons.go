package settings

import (
	"encoding/json"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func BuildSettingsButtons(
	localizer *localization.Localizer,
	settings []*BotSettings,
) [][]gotgbot.InlineKeyboardButton {
	buttons := make([]gotgbot.InlineKeyboardButton, 0, len(settings)+1)
	for _, setting := range settings {
		buttons = append(buttons, gotgbot.InlineKeyboardButton{
			Text:         localizer.T(&i18n.LocalizeConfig{MessageID: setting.ButtonKey}),
			CallbackData: "settings.options." + setting.ID,
		})
	}
	return util.ChunkedSlice(buttons, 2)
}

func BuildSettingsOptionsButtons(
	setting *BotSettings,
	res database.GetOrCreateChatRow,
	localizer *localization.Localizer,
) [][]gotgbot.InlineKeyboardButton {
	var buttons [][]gotgbot.InlineKeyboardButton

	switch setting.Type {
	case SettingsTypeToggle:
		var buttonText string
		value := setting.GetCurrentValueFunc(res)
		enabled, ok := value.(bool)
		if !ok {
			enabled = false
		}
		if enabled {
			buttonText = "✓ " + localizer.T(&i18n.LocalizeConfig{
				MessageID: localization.EnabledButton.ID,
			})
		} else {
			buttonText = "✗ " + localizer.T(&i18n.LocalizeConfig{
				MessageID: localization.DisabledButton.ID,
			})
		}
		buttons = append(buttons, []gotgbot.InlineKeyboardButton{
			{
				Text:         buttonText,
				CallbackData: "settings.toggle." + setting.ID,
			},
		})
	case SettingsTypeSelect:
		if setting.OptionsFunc != nil {
			options := setting.OptionsFunc(localizer)
			currentValue := setting.GetCurrentValueFunc(res)

			var optionButtons []gotgbot.InlineKeyboardButton

			for _, option := range options {
				var buttonText string

				if currentValue == option.Value {
					buttonText = "● " + option.Name
				} else {
					buttonText = "○ " + option.Name
				}
				valueBytes, err := json.Marshal(option.Value)
				if err != nil {
					continue
				}

				optionButtons = append(optionButtons, gotgbot.InlineKeyboardButton{
					Text:         buttonText,
					CallbackData: "settings.select." + setting.ID + "." + string(valueBytes),
				})
			}
			chunkSize := setting.OptionsChunk
			if chunkSize <= 0 {
				chunkSize = 1
			}
			buttons = append(buttons, util.ChunkedSlice(optionButtons, chunkSize)...)
		}
	}

	return buttons
}
