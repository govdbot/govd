package models

import (
	"context"
	"net/http"
	"regexp"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
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
	Context     context.Context
	Settings    *database.GetOrCreateChatRow
	HTTPClient  *networking.HTTPClient
	Config      *config.ExtractorConfig
}

func (e *ExtractorContext) SetSettings(settings *database.GetOrCreateChatRow) {
	e.Settings = settings
}

func (e *ExtractorContext) NewMedia() *Media {
	return &Media{
		ContentID:   e.ContentID,
		ContentURL:  e.ContentURL,
		ExtractorID: e.Extractor.ID,
	}
}

type ExtractorResponse struct {
	URL   string
	Media *Media
}

// peforms an HTTP request with the given method,
// url and params, using the extractor's HTTP client
func (ctx *ExtractorContext) Fetch(
	method string,
	url string,
	params *networking.RequestParams,
) (*http.Response, error) {
	if params == nil {
		params = &networking.RequestParams{}
	}
	return ctx.HTTPClient.FetchWithContext(
		ctx.Context,
		method,
		url,
		params,
	)
}

func (ctx *ExtractorContext) FetchLocation(
	url string,
	params *networking.RequestParams,
) (string, error) {
	resp, err := ctx.Fetch(
		http.MethodPost,
		url,
		params,
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	redirectURL := resp.Request.URL.String()
	return redirectURL, nil
}
