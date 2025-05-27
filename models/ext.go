package models

import (
	"regexp"

	"github.com/govdbot/govd/enums"
)

type Extractor struct {
	Name       string
	CodeName   string
	Type       enums.ExtractorType
	Category   enums.ExtractorCategory
	URLPattern *regexp.Regexp
	Host       []string
	IsDRM      bool
	IsRedirect bool
	IsHidden   bool

	Run func(*DownloadContext) (*ExtractorResponse, error)
}

type ExtractorResponse struct {
	MediaList []*Media
	URL       string // redirected URL
}

func (extractor *Extractor) NewMedia(
	contentID string,
	contentURL string,
) *Media {
	return &Media{
		ContentID:         contentID,
		ContentURL:        contentURL,
		ExtractorCodeName: extractor.CodeName,
	}
}
