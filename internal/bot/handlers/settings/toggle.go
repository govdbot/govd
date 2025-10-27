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

	chat, err := util.ChatFromContext(ctx)
	if err != nil {
		return err
	}
	isGroup := chat.Type == database.ChatTypeGroup

	localizer := localization.New(chat.Language)
	if isGroup && !util.CheckAdminPermission(b, ctx, localizer) {
		return nil
	}
	err = setting.ToggleFunc(
		context.Background(),
		chat.ChatID,
	)
	if err != nil {
		return err
	}

	return ListOptionsByID(b, ctx, settingID)
}
