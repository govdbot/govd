package extractors

import (
	"regexp"

	"github.com/govdbot/govd/internal/database"
)

type Extractor struct {
	ID          string
	DisplayName string

	URLPattern *regexp.Regexp
	Host       []string

	Hidden   bool
	Redirect bool

	GetFunc func() *Response
}

func (e *Extractor) NewMedia() *Media {
	return &Media{
		ExtractorID: e.ID,
	}
}

type Response struct {
	URL   string
	Media *Media
}

type Media struct {
	ContentID   string
	ContentURL  string
	ExtractorID string
	Caption     string
	NSFW        bool

	Items []*MediaItem
}

type MediaItem struct {
	Type     database.MediaType
	Duration int32
	Title    string
	Artist   string

	Formats []*MediaFormat
}

type MediaFormat struct {
	FormatID   string
	FileID     string
	AudioCodec database.MediaCodec
	VideoCodec database.MediaCodec
	Width      int32
	Height     int32
	Bitrate    int32

	URL          []string
	ThumbnailURL []string
}
