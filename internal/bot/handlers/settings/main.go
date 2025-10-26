package settings

import (
	"context"
	"strconv"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

var botSettings = []BotSettings{
	{
		ID:             "language",
		ButtonKey:      localization.LanguageButton.ID,
		DescriptionKey: localization.SelectLanguageMessage.ID,

		Type:  SettingsTypeSelect,
		Scope: SettingsScopeAll,

		OptionsFunc: func(_ *localization.Localizer) []*BotSettingsOptions {
			supportedLanguages := localization.B().LanguageTags()
			options := make([]*BotSettingsOptions, 0, len(supportedLanguages))

			for _, language := range supportedLanguages {
				languageCode := language.String()
				languageName := localization.New(languageCode).T(&i18n.LocalizeConfig{
					MessageID: localization.Language.ID,
				})
				options = append(options, &BotSettingsOptions{
					Name:  languageName,
					Value: languageCode,
				})
			}
			return options
		},
		SetValueFunc: func(ctx context.Context, chatID int64, value any) error {
			languageCode, ok := value.(string)
			if !ok {
				return nil
			}
			return database.Q().SetChatLanguage(ctx, database.SetChatLanguageParams{
				Language: languageCode,
				ChatID:   chatID,
			})
		},
		GetCurrentValueFunc: func(res *database.GetOrCreateChatRow) any {
			return res.Language
		},
		OptionsChunk: 3,
	},
	{
		ID:             "captions",
		ButtonKey:      localization.CaptionsButton.ID,
		DescriptionKey: localization.CaptionsSettingsMessage.ID,

		Type:  SettingsTypeToggle,
		Scope: SettingsScopeGroup,

		ToggleFunc: func(ctx context.Context, chatID int64) error {
			return database.Q().ToggleChatCaptions(ctx, chatID)
		},
		GetCurrentValueFunc: func(res *database.GetOrCreateChatRow) any {
			return res.Captions
		},
	},
	{
		ID:             "media_album",
		ButtonKey:      localization.MediaAlbumButton.ID,
		DescriptionKey: localization.MediaAlbumSettingsMessage.ID,

		Type:  SettingsTypeSelect,
		Scope: SettingsScopeGroup,

		OptionsFunc: func(l *localization.Localizer) []*BotSettingsOptions {
			limits := []int32{1, 5, 10, 15, 20}
			options := make([]*BotSettingsOptions, 0, len(limits))

			for _, limit := range limits {
				options = append(options, &BotSettingsOptions{
					Name:  strconv.Itoa(int(limit)),
					Value: limit,
				})
			}
			return options
		},
		SetValueFunc: func(ctx context.Context, chatID int64, value any) error {
			limit, ok := value.(int32)
			if !ok {
				if f, ok := value.(float64); ok {
					limit = int32(f)
				} else {
					return nil
				}
			}
			return database.Q().SetChatMediaAlbumLimit(ctx, database.SetChatMediaAlbumLimitParams{
				MediaAlbumLimit: limit,
				ChatID:          chatID,
			})
		},
		GetCurrentValueFunc: func(res *database.GetOrCreateChatRow) any {
			return res.MediaAlbumLimit
		},
		OptionsChunk: 5,
	},
	{
		ID:             "silent_mode",
		ButtonKey:      localization.SilentModeButton.ID,
		DescriptionKey: localization.SilentModeSettingsMessage.ID,

		Type:  SettingsTypeToggle,
		Scope: SettingsScopeGroup,

		ToggleFunc: func(ctx context.Context, chatID int64) error {
			return database.Q().ToggleChatSilentMode(ctx, chatID)
		},
		GetCurrentValueFunc: func(res *database.GetOrCreateChatRow) any {
			return res.Silent
		},
	},
	{
		ID:             "nsfw",
		ButtonKey:      localization.NsfwButton.ID,
		DescriptionKey: localization.NsfwSettingsMessage.ID,

		Type:  SettingsTypeToggle,
		Scope: SettingsScopeGroup,

		ToggleFunc: func(ctx context.Context, chatID int64) error {
			return database.Q().ToggleChatNsfw(ctx, chatID)
		},
		GetCurrentValueFunc: func(res *database.GetOrCreateChatRow) any {
			return res.Nsfw
		},
	},
	{
		ID:             "disabled_extractors",
		ButtonKey:      localization.DisabledExtractorsButton.ID,
		DescriptionKey: localization.DisabledExtractorsSettingsMessage.ID,

		Type:  SettingsTypeMany,
		Scope: SettingsScopeAll,

		OptionsFunc: func(l *localization.Localizer) []*BotSettingsOptions {
			options := make([]*BotSettingsOptions, 0, len(extractors.Extractors))
			for _, extractor := range extractors.Extractors {
				if extractor.Redirect || extractor.Hidden {
					continue
				}
				cfg := config.GetExtractorConfig(extractor.ID)
				if cfg != nil && cfg.IsDisabled {
					continue
				}
				options = append(options, &BotSettingsOptions{
					Name:  extractor.DisplayName,
					Value: extractor.ID,
				})
			}
			return options
		},
		AddValueFunc: func(ctx context.Context, chatID int64, value any) error {
			extractorID, ok := value.(string)
			if !ok {
				return nil
			}
			return database.Q().AddDisabledExtractor(ctx, database.AddDisabledExtractorParams{
				ExtractorID: extractorID,
				ChatID:      chatID,
			})
		},
		RemoveValueFunc: func(ctx context.Context, chatID int64, value any) error {
			extractorID, ok := value.(string)
			if !ok {
				return nil
			}
			return database.Q().RemoveDisabledExtractor(ctx, database.RemoveDisabledExtractorParams{
				ExtractorID: extractorID,
				ChatID:      chatID,
			})
		},
		GetCurrentValueFunc: func(res *database.GetOrCreateChatRow) any {
			return res.DisabledExtractors
		},
		OptionsChunk: 2,
	},
}

func SettingsHandler(b *gotgbot.Bot, ctx *ext.Context) error {
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

	var textMessage string

	scopeSettings := make([]*BotSettings, 0, len(botSettings))

	if isGroup {
		for _, setting := range botSettings {
			if setting.Scope == SettingsScopeAll || setting.Scope == SettingsScopeGroup {
				scopeSettings = append(scopeSettings, &setting)
			}
		}
		textMessage = localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.GroupSettingsMessage.ID,
		})
	} else {
		for _, setting := range botSettings {
			if setting.Scope == SettingsScopeAll || setting.Scope == SettingsScopePrivate {
				scopeSettings = append(scopeSettings, &setting)
			}
		}
		textMessage = localizer.T(&i18n.LocalizeConfig{
			MessageID: localization.PrivateSettingsMessage.ID,
		})
	}

	buttons := BuildSettingsButtons(localizer, scopeSettings)

	if isGroup {
		buttons = append(buttons, []gotgbot.InlineKeyboardButton{
			{
				Text: localizer.T(&i18n.LocalizeConfig{
					MessageID: localization.CloseButton.ID,
				}),
				CallbackData: "close",
			},
		})
	} else {
		buttons = append(buttons, []gotgbot.InlineKeyboardButton{
			{
				Text: localizer.T(&i18n.LocalizeConfig{
					MessageID: localization.BackButton.ID,
				}),
				CallbackData: "start",
			},
		})
	}

	if ctx.Message != nil {
		ctx.EffectiveMessage.Reply(
			b,
			textMessage,
			&gotgbot.SendMessageOpts{
				ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: buttons,
				},
			},
		)
	} else if ctx.CallbackQuery != nil {
		ctx.CallbackQuery.Answer(b, nil)
		ctx.EffectiveMessage.EditText(
			b,
			textMessage,
			&gotgbot.EditMessageTextOpts{
				ReplyMarkup: gotgbot.InlineKeyboardMarkup{
					InlineKeyboard: buttons,
				},
			},
		)
	}

	return nil
}

func GetSettingByID(settingID string) *BotSettings {
	for _, setting := range botSettings {
		if setting.ID == settingID {
			return &setting
		}
	}
	return nil
}
