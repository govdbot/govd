package m3u8

import (
	"fmt"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) processAlternatives(variants []*m3u8.Variant) []*models.MediaFormat {
	formats := make([]*models.MediaFormat, 0)
	seenAlternatives := make(map[string]bool)

	for _, variant := range variants {
		if variant == nil {
			continue
		}
		for _, alt := range variant.Alternatives {
			if alt == nil || alt.GroupId == "" || seenAlternatives[alt.GroupId] {
				continue
			}
			seenAlternatives[alt.GroupId] = true
			format := p.parseAlternative(variants, alt)
			if format != nil {
				formats = append(formats, format)
			}
		}
	}

	return formats
}

func (p *M3U8Parser) parseAlternative(variants []*m3u8.Variant, alt *m3u8.Alternative) *models.MediaFormat {
	if alt == nil || alt.URI == "" || alt.Type != "AUDIO" {
		return nil
	}
	altURL := p.resolveURL(alt.URI)
	audioCodec := getAudioAlternativeCodec(variants, alt)

	format := &models.MediaFormat{
		FormatID:   "hls-" + alt.GroupId,
		Type:       database.MediaTypeAudio,
		AudioCodec: audioCodec,
		URL:        []string{altURL},
	}

	altFormats, err := p.ParseAlternative(altURL)
	if err != nil || len(altFormats) == 0 {
		return format
	}

	enrichFormatFromVariant(format, altFormats[0])

	return format
}

func (p *M3U8Parser) ParseAlternative(altURL string) ([]*models.MediaFormat, error) {
	data, err := ParseM3U8FromURL(p.Context, altURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse alternative M3U8: %w", err)
	}
	return data, nil
}

func getAudioAlternativeCodec(variants []*m3u8.Variant, alt *m3u8.Alternative) database.MediaCodec {
	if alt == nil || alt.URI == "" || alt.Type != "AUDIO" {
		return ""
	}
	for _, variant := range variants {
		if variant == nil || variant.URI == "" || variant.Audio != alt.GroupId {
			continue
		}
		if audioCodec := util.ParseAudioCodec(variant.Codecs); audioCodec != "" {
			return audioCodec
		}
	}
	return ""
}
