package localization

import (
	"golang.org/x/text/language"
)

func GetLocaleFromCode(code string, fallback string) string {
	if IsCodeSupported(code) {
		return code
	}
	if IsCodeSupported(fallback) {
		return fallback
	}
	return language.English.String()
}

func IsCodeSupported(code string) bool {
	tags := bundle.LanguageTags()
	for _, tag := range tags {
		if tag.String() == code {
			return true
		}
	}
	return false
}
