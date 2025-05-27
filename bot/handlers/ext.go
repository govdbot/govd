package handlers

import (
	"fmt"
	"strings"

	"github.com/govdbot/govd/config"
	extractors "github.com/govdbot/govd/ext"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func ExtractorsHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	ctx.CallbackQuery.Answer(bot, nil)

	messageText := "available extractors:\n"
	extractorNames := make([]string, 0, len(extractors.List))
	for _, extractor := range extractors.List {
		if extractor.IsRedirect || extractor.IsHidden {
			continue
		}
		cfg := config.GetExtractorConfig(extractor)
		if cfg != nil && cfg.IsDisabled {
			extractorNames = append(extractorNames, fmt.Sprintf(
				"<s>%s</s> <i>(disabled)</i>",
				extractor.Name,
			))
		} else {
			extractorNames = append(extractorNames, extractor.Name)
		}
	}
	messageText += strings.Join(extractorNames, ", ")

	ctx.EffectiveMessage.EditText(
		bot,
		messageText,
		&gotgbot.EditMessageTextOpts{
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled: true,
			},
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{
						{
							Text:         "back",
							CallbackData: "start",
						},
					},
				},
			},
		},
	)
	return nil
}
