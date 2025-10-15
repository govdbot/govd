package m3u8

import (
	"fmt"

	"github.com/govdbot/govd/internal/models"
	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) parseMasterPlaylist(playlist *m3u8.MasterPlaylist) ([]*models.MediaFormat, error) {
	if len(playlist.Variants) == 0 {
		return nil, fmt.Errorf("no variants found in master playlist")
	}

	formats := make([]*models.MediaFormat, 0)

	altFormats := p.processAlternatives(playlist.Variants)
	formats = append(formats, altFormats...)

	variantFormats, err := p.processVariants(playlist.Variants)
	if err != nil {
		return nil, fmt.Errorf("failed processing variants: %w", err)
	}
	formats = append(formats, variantFormats...)

	return formats, nil
}
