package localization

import (
	"golang.org/x/text/language"
)

func GetLocaleFromCode(code string) string {
	tags := bundle.LanguageTags()
	for _, tag := range tags {
		if tag.String() == code {
			return code
		}
	}
	return language.English.String()
}
