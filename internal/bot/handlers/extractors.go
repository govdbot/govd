package handlers

import (
	"slices"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func ExtractorsHandler(bot *gotgbot.Bot, ctx *ext.Context) error {
	chat, err := util.ChatFromContext(ctx)
	if err != nil {
		return err
	}
	ctx.CallbackQuery.Answer(bot, nil)
	localizer := localization.New(chat.Language)
	text := localizer.T(&i18n.LocalizeConfig{
		MessageID: localization.SupportedExtractorsMessage.ID,
	})
	for _, e := range getSupportedExtractors() {
		text += "\n• " + e
	}
	ctx.EffectiveMessage.EditText(
		bot, text,
		&gotgbot.EditMessageTextOpts{
			ReplyMarkup: gotgbot.InlineKeyboardMarkup{
				InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
					{
						{
							Text: localizer.T(&i18n.LocalizeConfig{
								MessageID: localization.BackButton.ID,
							}),
							CallbackData: "start",
						},
					},
				},
			},
		},
	)
	return nil
}

func getSupportedExtractors() []string {
	extractorNames := make([]string, 0, len(extractors.Extractors))
	for _, e := range extractors.Extractors {
		if e.Hidden || e.Redirect {
			continue
		}
		cfg := config.GetExtractorConfig(e.ID)
		if cfg != nil && cfg.IsDisabled {
			continue
		}
		extractorNames = append(extractorNames, e.DisplayName)
	}
	slices.Sort(extractorNames)
	return extractorNames
}
