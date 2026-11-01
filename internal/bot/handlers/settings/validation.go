package settings

import (
	"bytes"
	"encoding/json"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
)

func allowedScope(s *BotSettings, chat *database.GetOrCreateChatRow) bool {
	if s == nil || chat == nil {
		return false
	}
	if s.Scope == SettingsScopeAll {
		return true
	}
	if chat.Type == database.ChatTypeGroup {
		return s.Scope == SettingsScopeGroup
	}
	return s.Scope == SettingsScopePrivate
}
func allowedValue(s *BotSettings, l *localization.Localizer, value any) bool {
	if s.OptionsFunc == nil {
		return false
	}
	want, err := json.Marshal(value)
	if err != nil {
		return false
	}
	for _, option := range s.OptionsFunc(l) {
		v, e := json.Marshal(option.Value)
		if e == nil && bytes.Equal(v, want) {
			return true
		}
	}
	return false
}
