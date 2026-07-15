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

func GetHLSFormatsFromURL(
	ctx *models.ExtractorContext,
	hlsURL string,
	duration int32,
) ([]*models.MediaFormat, error) {
	if hlsURL == "" {
		return nil, nil
	}
	formats, err := m3u8.ParseM3U8FromURL(ctx, hlsURL, nil)
	if err != nil {
		return nil, err
	}
	for _, f := range formats {
		if duration > 0 {
			f.Duration = duration
		}
	}
	return formats, nil
}

func GetDASHFormatsFromURL(
	ctx *models.ExtractorContext,
	dashURL string,
	duration int32,
) ([]*models.MediaFormat, error) {
	// Reddit DASH manifest is MPD XML - not yet fully parsed in govd
	// For now, rely on HLS and fallback. Return empty to avoid breaking.
	// Future: add MPD parser similar to HLS.
	// We attempt to fetch via HLS derived URL as best effort fallback.
	if dashURL == "" {
		return nil, nil
	}
	// Attempt to derive HLS from DASH video ID as fallback
	if m := videoURLPattern.FindStringSubmatch(dashURL); len(m) >= 2 {
		videoID := m[1]
		hlsURL := fmt.Sprintf(hlsURLFormat, videoID)
		formats, err := m3u8.ParseM3U8FromURL(ctx, hlsURL, nil)
		if err == nil {
			for _, f := range formats {
				if duration > 0 {
					f.Duration = duration
				}
			}
			return formats, nil
		}
	}
	return nil, nil
}
