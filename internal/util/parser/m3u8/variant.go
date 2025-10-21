package m3u8

import (
	"fmt"
	"sync"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) processVariants(variants []*m3u8.Variant) ([]*models.MediaFormat, error) {
	var wg sync.WaitGroup

	variants = getValidVariants(variants)
	if len(variants) == 0 {
		return nil, fmt.Errorf("no valid variants found in master playlist")
	}

	results := make(chan *models.MediaFormat, len(variants))
	semaphore := make(chan struct{}, MAX_CONCURRENT_REQUESTS)

	wg.Add(len(variants))
	for _, variant := range variants {
		go func(v *m3u8.Variant) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			p.parseVariant(results, v)
		}(variant)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	formats := make([]*models.MediaFormat, 0, len(variants))
	for format := range results {
		formats = append(formats, format)
	}

	return formats, nil
}

func (p *M3U8Parser) parseVariant(results chan<- *models.MediaFormat, variant *m3u8.Variant) {
	width, height := parseResolution(variant.Resolution)
	mediaType, videoCodec, audioCodec := parseVariantType(variant)
	variantURL := p.resolveURL(variant.URI)

	if variant.Audio != "" {
		audioCodec = ""
	}

	format := &models.MediaFormat{
		FormatID:   fmt.Sprintf("hls-%d", variant.Bandwidth/1000),
		Type:       mediaType,
		VideoCodec: videoCodec,
		AudioCodec: audioCodec,
		Bitrate:    int64(variant.Bandwidth),
		Width:      width,
		Height:     height,
		URL:        []string{variantURL},
	}

	data, err := p.ParseVariant(variantURL)
	if err != nil {
		results <- format
		return
	}

	if len(data) > 0 {
		enrichFormatFromVariant(format, data[0])
	}

	results <- format
}

func (p *M3U8Parser) ParseVariant(variantURL string) ([]*models.MediaFormat, error) {
	data, err := ParseM3U8FromURL(p.Context, variantURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse variant M3U8: %w", err)
	}
	return data, nil
}

func parseVariantType(variant *m3u8.Variant) (
	database.MediaType,
	database.MediaCodec,
	database.MediaCodec,
) {
	videoCodec := util.ParseVideoCodec(variant.Codecs)
	audioCodec := util.ParseAudioCodec(variant.Codecs)

	var mediaType database.MediaType
	switch {
	case videoCodec != "":
		mediaType = database.MediaTypeVideo
	case audioCodec != "":
		mediaType = database.MediaTypeAudio
	}

	return mediaType, videoCodec, audioCodec
}

func enrichFormatFromVariant(dst, src *models.MediaFormat) {
	if src.Segments != nil {
		dst.Segments = src.Segments
	}
	if src.InitSegment != "" {
		dst.InitSegment = src.InitSegment
	}
	if src.Duration > 0 {
		dst.Duration = src.Duration
	}
	if src.DecryptionKey != nil {
		dst.DecryptionKey = src.DecryptionKey
	}
}
