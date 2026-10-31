package settings

import (
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"testing"
)

func TestManipulatedCallbackKindsReturnBeforeDatabase(t *testing.T) {
	for _, data := range []string{"settings.toggle.language", "settings.select.captions.true", "settings.many.language.add.x", "settings.toggle.unknown"} {
		ctx := &ext.Context{Update: &gotgbot.Update{CallbackQuery: &gotgbot.CallbackQuery{Data: data}}}
		if len(data) >= 16 && data[:16] == "settings.toggle." {
			if err := SettingsToggleHandler(nil, ctx); err != nil {
				t.Fatal(err)
			}
		} else if len(data) >= 16 && data[:16] == "settings.select." {
			if err := SettingsSelectHandler(nil, ctx); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := SettingsManyHandler(nil, ctx); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestSettingsScopeAndValues(t *testing.T) {
	s := GetSettingByID("media_album")
	private := &database.GetOrCreateChatRow{Type: database.ChatTypePrivate}
	group := &database.GetOrCreateChatRow{Type: database.ChatTypeGroup}
	if allowedScope(s, private) || !allowedScope(s, group) {
		t.Fatal("scope check failed")
	}
	l := localization.New("en")
	if !allowedValue(s, l, float64(5)) || allowedValue(s, l, float64(6)) || allowedValue(s, l, float64(1.5)) || allowedValue(s, l, "5") {
		t.Fatal("value validation failed")
	}
}
