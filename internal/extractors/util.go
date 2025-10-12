package extractors

import (
	"context"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

const maxRedirects = 5

var extractorsByHost = getExtractorsMap()

func FromURL(ctx context.Context, url string) *models.ExtractorContext {
	var redirectCount int

	currentURL := url

	for redirectCount <= maxRedirects {
		host, err := util.ExtractBaseHost(currentURL)
		if err != nil {
			return nil
		}

		extractors := getExtractorsByHost(host)
		if len(extractors) == 0 {
			return nil
		}

		var extractor *models.Extractor
		var matches []string
		var groups map[string]string

		for _, e := range extractors {
			matches = e.URLPattern.FindStringSubmatch(currentURL)
			if matches != nil {
				extractor = e
				groups = util.GetNamedGroups(e.URLPattern, currentURL)
				break
			}
		}

		if extractor == nil || matches == nil {
			return nil
		}

		cfg := config.GetExtractorConfig(extractor.ID)
		if cfg.IsDisabled {
			return nil
		}

		extractorCtx := &models.ExtractorContext{
			ContentID:   groups["id"],
			ContentURL:  groups["match"],
			MatchGroups: groups,
			Extractor:   extractor,
			Context:     ctx,
			Config:      cfg,
			HTTPClient: networking.NewHTTPClient(
				&networking.NewHTTPClientOptions{
					Cookies:   util.GetExtractorCookies(extractor.ID),
					EdgeProxy: cfg.EdgeProxy,
					Proxy:     cfg.Proxy,
				},
			),
		}
		if !extractor.Redirect {
			return extractorCtx
		}

		// extractor requires fetching the URL for redirection
		logger.L.Debugf("following redirect for extractor: %s", extractor.ID)

		response, err := extractor.GetFunc(extractorCtx)
		if err != nil {
			logger.L.Errorf("%s: %v", extractor.ID, err)
			return nil
		}
		if response.URL == "" {
			logger.L.Errorf("%s: no URL found in response", extractor.ID)
			return nil
		}

		currentURL = response.URL
		redirectCount++

		if redirectCount > maxRedirects {
			logger.L.Errorf("%s: exceeded maximum number of redirects (%d)", extractor.ID, maxRedirects)
			return nil
		}
	}
	return nil
}

func getExtractorsMap() map[string][]*models.Extractor {
	extractorsByHost := make(map[string][]*models.Extractor)
	for _, extractor := range Extractors {
		if len(extractor.Host) == 0 {
			continue
		}
		for _, domain := range extractor.Host {
			extractorsByHost[domain] = append(extractorsByHost[domain], extractor)
		}
	}
	return extractorsByHost
}

func getExtractorsByHost(host string) []*models.Extractor {
	return extractorsByHost[host]
}
