package models

import (
	"net/http"
	"regexp"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/networking"
)

type Extractor struct {
	ID          string
	DisplayName string

	URLPattern *regexp.Regexp
	Host       []string

	Hidden   bool
	Redirect bool

	GetFunc func(*ExtractorContext) (*ExtractorResponse, error)
}

type ExtractorContext struct {
	ContentURL  string
	ContentID   string
	MatchGroups map[string]string
	Extractor   *Extractor

	HTTPClient *networking.HTTPClient
	Config     *config.ExtractorConfig
	Cookies    []*http.Cookie
}

func (e *ExtractorContext) NewMedia() *Media {
	return &Media{
		ExtractorID: e.Extractor.ID,
	}
}

type ExtractorResponse struct {
	URL   string
	Media *Media
}

// peforms an HTTP request with the given method, url and params,
// using the extractor's HTTP client and cookies
func (ctx *ExtractorContext) Fetch(
	method string,
	url string,
	params *networking.RequestParams,
) (*http.Response, error) {
	if params == nil {
		params = &networking.RequestParams{}
	}
	params.Cookies = ctx.Cookies
	return ctx.HTTPClient.Fetch(
		method,
		url,
		params,
	)
}
