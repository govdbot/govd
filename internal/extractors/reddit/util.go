package reddit

import (
	"fmt"
	"regexp"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util/parser/m3u8"
)

const hlsURLFormat = "https://v.redd.it/%s/HLSPlaylist.m3u8"

var videoURLPattern = regexp.MustCompile(`https?://v\.redd\.it/([^/]+)`)

func GetHLSFormats(
	ctx *models.ExtractorContext,
	videoURL string,
	duration int32,
) ([]*models.MediaFormat, error) {
	matches := videoURLPattern.FindStringSubmatch(videoURL)
	if len(matches) < 2 {
		return nil, nil
	}

	videoID := matches[1]
	hlsURL := fmt.Sprintf(hlsURLFormat, videoID)

	formats, err := m3u8.ParseM3U8FromURL(ctx, hlsURL, nil)
	if err != nil {
		return nil, err
	}

	for _, format := range formats {
		format.Duration = duration
	}

	return formats, nil
}
