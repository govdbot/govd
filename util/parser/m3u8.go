package parser

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"

	"github.com/grafov/m3u8"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func ParseM3U8Content(
	content []byte,
	baseURL string,
) ([]*models.MediaFormat, error) {
	baseURLObj, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	buf := bytes.NewBuffer(content)
	playlist, listType, err := m3u8.DecodeFrom(buf, false)
	if err != nil {
		return nil, fmt.Errorf("failed parsing m3u8: %w", err)
	}

	switch listType {
	case m3u8.MASTER:
		zap.S().Debugf("detected master playlist")
		master, ok := playlist.(*m3u8.MasterPlaylist)
		if !ok {
			return nil, errors.New("failed to cast to master playlist")
		}
		return parseMasterPlaylist(master, baseURLObj)
	case m3u8.MEDIA:
		zap.S().Debugf("detected media playlist")
		media, ok := playlist.(*m3u8.MediaPlaylist)
		if !ok {
			return nil, errors.New("failed to cast to media playlist")
		}
		return parseMediaPlaylist(media, baseURLObj)
	}

	return nil, errors.New("unsupported m3u8 playlist type")
}

func parseMasterPlaylist(
	playlist *m3u8.MasterPlaylist,
	baseURL *url.URL,
) ([]*models.MediaFormat, error) {
	// preallocate formats with a capacity based
	// on the number of variants. each variant can
	// potentially create one format, and some
	// alternatives might add more
	formats := make([]*models.MediaFormat, 0, len(playlist.Variants)*2)

	seenAlternatives := make(map[string]bool)
	for _, variant := range playlist.Variants {
		if variant == nil || variant.URI == "" {
			continue
		}
		for _, alt := range variant.Alternatives {
			if _, ok := seenAlternatives[alt.GroupId]; ok {
				continue
			}
			seenAlternatives[alt.GroupId] = true
			format := parseAlternative(
				playlist.Variants,
				alt, baseURL,
			)
			if format == nil {
				continue
			}
			formats = append(formats, format)
		}
		width, height := getResolution(variant.Resolution)
		mediaType, videoCodec, audioCodec := parseVariantType(variant)
		variantURL := resolveURL(baseURL, variant.URI)
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
		variantContent, err := fetchContent(variantURL)
		if err == nil {
			variantFormats, err := ParseM3U8Content(variantContent, variantURL)
			if err == nil && len(variantFormats) > 0 {
				format.Segments = variantFormats[0].Segments
				if variantFormats[0].Duration > 0 {
					format.Duration = variantFormats[0].Duration
				}
				formats = append(formats, format)
			} else {
				zap.S().Warnf("skipping variant due to: %v", err)
			}
		}
	}
	return formats, nil
}

func parseMediaPlaylist(
	playlist *m3u8.MediaPlaylist,
	baseURL *url.URL,
) ([]*models.MediaFormat, error) {
	initialCapacity := len(playlist.Segments)
	if playlist.Map != nil && playlist.Map.URI != "" {
		initialCapacity++
	}
	segments := make([]string, 0, initialCapacity)

	var totalDuration float64
	initSegment := playlist.Map
	if initSegment != nil && initSegment.URI != "" {
		initSegmentURL := resolveURL(baseURL, initSegment.URI)
		segments = append(segments, initSegmentURL)
	}
	for _, segment := range playlist.Segments {
		if segment != nil && segment.URI != "" {
			segmentURL := resolveURL(baseURL, segment.URI)
			segments = append(segments, segmentURL)
			totalDuration += segment.Duration
			if segment.Limit > 0 {
				// byterange not supported
				break
			}
		}
	}
	format := &models.MediaFormat{
		FormatID: "hls",
		Duration: int64(totalDuration),
		URL:      []string{baseURL.String()},
		Segments: segments,
	}
	return []*models.MediaFormat{format}, nil
}

func parseAlternative(
	variants []*m3u8.Variant,
	alternative *m3u8.Alternative,
	baseURL *url.URL,
) *models.MediaFormat {
	if alternative == nil || alternative.URI == "" {
		return nil
	}
	if alternative.Type != "AUDIO" {
		return nil
	}
	altURL := resolveURL(baseURL, alternative.URI)
	audioCodec := getAudioAlternativeCodec(variants, alternative)
	format := &models.MediaFormat{
		FormatID:   "hls" + alternative.GroupId,
		Type:       enums.MediaTypeAudio,
		AudioCodec: audioCodec,
		URL:        []string{altURL},
	}
	altContent, err := fetchContent(altURL)
	if err == nil {
		altFormats, err := ParseM3U8Content(altContent, altURL)
		if err == nil && len(altFormats) > 0 {
			format.Segments = altFormats[0].Segments
			if altFormats[0].Duration > 0 {
				format.Duration = altFormats[0].Duration
			}
		} else {
			zap.S().Warnf("skipping alternative due to: %v", err)
			return nil
		}
	}
	return format
}

func getAudioAlternativeCodec(
	variants []*m3u8.Variant,
	alt *m3u8.Alternative,
) enums.MediaCodec {
	if alt == nil || alt.URI == "" {
		return ""
	}
	if alt.Type != "AUDIO" {
		return ""
	}
	for _, variant := range variants {
		if variant == nil || variant.URI == "" {
			continue
		}
		if variant.Audio != alt.GroupId {
			continue
		}
		audioCodec := getAudioCodec(variant.Codecs)
		if audioCodec != "" {
			return audioCodec
		}
	}
	return ""
}

func ParseM3U8FromURL(url string) ([]*models.MediaFormat, error) {
	body, err := fetchContent(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch m3u8 content: %w", err)
	}
	return ParseM3U8Content(body, url)
}

func fetchContent(url string) ([]byte, error) {
	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status code: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func getResolution(
	resolution string,
) (int64, int64) {
	var width, height int
	if _, err := fmt.Sscanf(resolution, "%dx%d", &width, &height); err == nil {
		return int64(width), int64(height)
	}
	return 0, 0
}

func parseVariantType(
	variant *m3u8.Variant,
) (enums.MediaType, enums.MediaCodec, enums.MediaCodec) {
	var mediaType enums.MediaType
	var videoCodec, audioCodec enums.MediaCodec

	videoCodec = getVideoCodec(variant.Codecs)
	audioCodec = getAudioCodec(variant.Codecs)

	if videoCodec != "" {
		mediaType = enums.MediaTypeVideo
	} else if audioCodec != "" {
		mediaType = enums.MediaTypeAudio
	}

	return mediaType, videoCodec, audioCodec
}

func getVideoCodec(codecs string) enums.MediaCodec {
	switch {
	case strings.Contains(codecs, "avc"), strings.Contains(codecs, "h264"):
		return enums.MediaCodecAVC
	case strings.Contains(codecs, "hvc"), strings.Contains(codecs, "h265"):
		return enums.MediaCodecHEVC
	case strings.Contains(codecs, "av01"):
		return enums.MediaCodecAV1
	case strings.Contains(codecs, "vp9"):
		return enums.MediaCodecVP9
	case strings.Contains(codecs, "vp8"):
		return enums.MediaCodecVP8
	default:
		return ""
	}
}

func getAudioCodec(codecs string) enums.MediaCodec {
	switch {
	case strings.Contains(codecs, "mp4a"):
		return enums.MediaCodecAAC
	case strings.Contains(codecs, "opus"):
		return enums.MediaCodecOpus
	case strings.Contains(codecs, "mp3"):
		return enums.MediaCodecMP3
	case strings.Contains(codecs, "flac"):
		return enums.MediaCodecFLAC
	case strings.Contains(codecs, "vorbis"):
		return enums.MediaCodecVorbis
	default:
		return ""
	}
}

func resolveURL(base *url.URL, uri string) string {
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return uri
	}
	ref, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return base.ResolveReference(ref).String()
}
