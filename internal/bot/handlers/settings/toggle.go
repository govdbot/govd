package settings

import (
	"context"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
)

func SettingsToggleHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	// settings.toggle.id
	parts := strings.Split(ctx.CallbackQuery.Data, ".")
	if len(parts) < 3 {
		return nil
	}
	settingID := parts[2]

	setting := GetSettingByID(settingID)
	if setting == nil {
		return nil
	}

	chat := ctx.EffectiveChat
	isGroup := chat.Type != gotgbot.ChatTypePrivate
	user := ctx.EffectiveUser

	var chatType database.ChatType
	if isGroup {
		chatType = database.ChatTypeGroup
	} else {
		chatType = database.ChatTypePrivate
	}

	res, err := database.Q().GetOrCreateChat(
		context.Background(),
		database.GetOrCreateChatParams{
			ChatID:   chat.Id,
			Type:     chatType,
			Language: localization.GetLocaleFromCode(user.LanguageCode),
		},
	)
	if err != nil {
		return err
	}
	localizer := localization.New(res.Language)
	if isGroup && !util.CheckAdminPermission(b, ctx, localizer) {
		return nil
	}
	err = setting.ToggleFunc(
		context.Background(),
		chat.Id,
	)
	if err != nil {
		return err
	}

	return ListOptionsByID(b, ctx, settingID)
}
