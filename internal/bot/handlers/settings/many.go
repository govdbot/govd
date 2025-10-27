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

func SettingsManyHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	// settings.many.id.action.value
	parts := strings.Split(ctx.CallbackQuery.Data, ".")
	if len(parts) < 5 {
		return nil
	}
	settingID := parts[2]
	action := parts[3]
	value := parts[4]

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

	switch action {
	case "add":
		if setting.AddValueFunc != nil {
			err = setting.AddValueFunc(
				context.Background(),
				chat.ChatID, value,
			)
			if err != nil {
				return err
			}
		}
	case "remove":
		if setting.RemoveValueFunc != nil {
			err = setting.RemoveValueFunc(
				context.Background(),
				chat.ChatID, value,
			)
			if err != nil {
				return err
			}
		}
	default:
		return nil
	}

	return ListOptionsByID(b, ctx, settingID)
}
