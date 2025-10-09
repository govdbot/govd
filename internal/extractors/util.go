package extractors

import (
	"sync"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

const (
	maxRedirects = 5
)

var (
	extractorsByHost  map[string][]*models.Extractor
	extractorsMapOnce sync.Once
)

func FromURL(url string) *models.ExtractorContext {
	var redirectCount int

	currentURL := url

	for redirectCount <= maxRedirects {
		host, err := util.ExtractBaseHost(currentURL)
		if err != nil {
			return nil
		}
		extractors := GetExtractorsByHost(host)
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

		// cfg := config.GetExtractorConfig(extractor)
		// if cfg != nil && cfg.IsDisabled {
		// 	return nil
		// }

		ctx := &models.ExtractorContext{
			ContentID:   groups["id"],
			ContentURL:  groups["match"],
			MatchGroups: groups,
			Extractor:   extractor,
		}
		if !extractor.Redirect {
			return ctx
		}

		// extractor requires fetching the URL for redirection
		response, err := extractor.GetFunc(ctx)
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

func initExtractorsMap() {
	extractorsMapOnce.Do(func() {
		extractorsByHost = make(map[string][]*models.Extractor)
		for _, extractor := range Extractors {
			if len(extractor.Host) == 0 {
				continue
			}
			for _, domain := range extractor.Host {
				extractorsByHost[domain] = append(extractorsByHost[domain], extractor)
			}
		}
	})
}

func GetExtractorsByHost(host string) []*models.Extractor {
	initExtractorsMap()

	return extractorsByHost[host]
}
