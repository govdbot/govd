package settings

import (
	"context"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
)

type SettingsScope string

const (
	SettingsScopePrivate SettingsScope = "private"
	SettingsScopeGroup   SettingsScope = "group"
	SettingsScopeAll     SettingsScope = "all"
)

type SettingsType string

const (
	SettingsTypeSelect SettingsType = "select"
	SettingsTypeToggle SettingsType = "toggle"
)

type BotSettings struct {
	ID             string
	ButtonKey      string
	DescriptionKey string

	Type  SettingsType
	Scope SettingsScope

	OptionsFunc         func(*localization.Localizer) []*BotSettingsOptions
	GetCurrentValueFunc func(database.GetOrCreateChatRow) any

	ToggleFunc   func(context.Context, int64) error
	SetValueFunc func(context.Context, int64, any) error

	OptionsChunk int
}

type BotSettingsOptions struct {
	Name  string
	Value any
}
