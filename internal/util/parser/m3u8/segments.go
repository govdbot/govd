package m3u8

import (
	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) extractSegments(playlist *m3u8.MediaPlaylist) ([]string, string, float64) {
	segments := make([]string, 0, len(playlist.Segments))

	var totalDuration float64
	var initSegment string

	if playlist.Map != nil && playlist.Map.URI != "" {
		initSegment = p.resolveURL(playlist.Map.URI)
	}

	for _, segment := range playlist.Segments {
		if segment == nil || segment.URI == "" {
			continue
		}

		segmentURL := p.resolveURL(segment.URI)
		segments = append(segments, segmentURL)
		totalDuration += segment.Duration

		if segment.Limit > 0 {
			break
		}
	}
	return segments, initSegment, totalDuration
}
