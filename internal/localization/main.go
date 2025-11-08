package localization

import (
	"embed"

	"github.com/BurntSushi/toml"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/active.*.toml
var LocaleFS embed.FS

var bundle = i18n.NewBundle(language.English)

func Init() {
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
	mustLoad("locales/active.en.toml")
	mustLoad("locales/active.it.toml")
	mustLoad("locales/active.es.toml")
	mustLoad("locales/active.fr.toml")
	mustLoad("locales/active.de.toml")
	mustLoad("locales/active.pt.toml")
	mustLoad("locales/active.ru.toml")
	mustLoad("locales/active.id.toml")
	mustLoad("locales/active.tr.toml")
	mustLoad("locales/active.uk.toml")
	mustLoad("locales/active.ar.toml")

}

func mustLoad(file string) {
	_, err := bundle.LoadMessageFileFS(LocaleFS, file)
	if err != nil {
		panic(err)
	}
}

func B() *i18n.Bundle {
	return bundle
}
