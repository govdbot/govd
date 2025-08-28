package localization

import (
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

type Localizer struct {
	L *i18n.Localizer
}

func New(language string) *Localizer {
	return &Localizer{
		L: i18n.NewLocalizer(bundle, language),
	}
}

func (loc Localizer) T(cfg *i18n.LocalizeConfig) string {
	s, err := loc.L.Localize(cfg)
	if err != nil {
		// fallback to english
		en := i18n.NewLocalizer(bundle, "en")
		s, err = en.Localize(cfg)
		if err != nil {
			return cfg.MessageID
		}
		return s
	}
	return s
}
