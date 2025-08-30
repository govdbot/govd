package tiktok

import (
	"regexp"

	"github.com/govdbot/govd/internal/extractors"
)

var Extractor = &extractors.Extractor{
	ID:          "tiktok",
	DisplayName: "TikTok",

	URLPattern: regexp.MustCompile(`https?:\/\/((www|m)\.)?(vx)?tiktok\.com\/((?:embed|@[\w\.-]*)\/)?(v(ideo)?|p(hoto)?)\/(?P<id>[0-9]+)`),
	Host:       []string{"tiktok", "vxtiktok"},

	GetFunc: func() *extractors.Response {
		// TODO
		return nil
	},
}
