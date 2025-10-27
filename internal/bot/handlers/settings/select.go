package settings

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/util"
)

func SettingsSelectHandler(b *gotgbot.Bot, ctx *ext.Context) error {
	// settings.select.id.value
	parts := strings.Split(ctx.CallbackQuery.Data, ".")
	if len(parts) < 4 {
		return nil
	}
	settingID := parts[2]
	valueStr := parts[3]

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
	var value any
	err = json.Unmarshal([]byte(valueStr), &value)
	if err != nil {
		return err
	}
	if setting.SetValueFunc != nil {
		err = setting.SetValueFunc(
			context.Background(),
			chat.ChatID, value,
		)
		if err != nil {
			return err
		}
	}
	return ListOptionsByID(b, ctx, settingID)
}
