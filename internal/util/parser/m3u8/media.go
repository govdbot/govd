package m3u8

import (
	"github.com/govdbot/govd/internal/models"
	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) parseMediaPlaylist(playlist *m3u8.MediaPlaylist) ([]*models.MediaFormat, error) {
	segments, initSegment, totalDuration := p.extractSegments(playlist)

	format := &models.MediaFormat{
		FormatID:    "hls",
		Duration:    int32(totalDuration),
		URL:         []string{p.BaseURL.String()},
		Segments:    segments,
		InitSegment: initSegment,
	}

	if err := p.handleEncryption(playlist, format); err != nil {
		return nil, err
	}

	return []*models.MediaFormat{format}, nil
}
