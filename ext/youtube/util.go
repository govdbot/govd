package youtube

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"
)

const invEndpoint = "/api/v1/videos/"

var invInstance string


func ParseInvFormats(data *InvResponse) []*models.MediaFormat {
	// Decided to use FormatStream instead of AdaptiveFormats since this latter might not be available in some videos.
        //formats := make([]*models.MediaFormat, 0, len(data.AdaptiveFormats))
	formats := make([]*models.MediaFormat, 0, len(data.FormatStreams))
	duration := data.LengthSeconds
	
	// As explained above, using FormatStreams
	//for _, format := range data.AdaptiveFormats {
	for _, format := range data.FormatStreams {
		if format.URL == "" {
			continue
		}
		mediaType, vCodec, aCodec := ParseStreamType(format.Type)
		if mediaType == "" {
			continue
		}
		var bitrate int64
		if format.Bitrate != "" {
			bitrate, _ = strconv.ParseInt(format.Bitrate, 10, 64)
		}
		var width, height int64
		if format.Size != "" {
			dimensions := strings.Split(format.Size, "x")
			if len(dimensions) == 2 {
				width, _ = strconv.ParseInt(dimensions[0], 10, 64)
				height, _ = strconv.ParseInt(dimensions[1], 10, 64)
			}
		}
		// we dont use thumbnails provided by youtube
		// due to black bars on the sides for some videos
		formats = append(formats, &models.MediaFormat{
			Type:       mediaType,
			VideoCodec: vCodec,
			AudioCodec: aCodec,
			FormatID:   format.Itag,
			Width:      width,
			Height:     height,
			Bitrate:    bitrate,
			Duration:   int64(duration),
			URL:        []string{ParseInvURL(format.URL)},
			Title:      data.Title,
			Artist:     data.Author,
			DownloadConfig: &models.DownloadConfig{
				// youtube throttles the download speed
				// if chunk size is too small
				ChunkSize: 10 * 1024 * 1024, // 10 MB
			},
		})
	}
	return formats
}

func ParseStreamType(
	streamType string,
) (enums.MediaType, enums.MediaCodec, enums.MediaCodec) {
	parts := strings.Split(streamType, "; ")
	if len(parts) != 2 {
		// unknown stream type
		return "", "", ""
	}
	codecs := parts[1]

	var mediaType enums.MediaType
	var videoCodec, audioCodec enums.MediaCodec

	videoCodec = ParseVideoCodec(codecs)
	audioCodec = ParseAudioCodec(codecs)

	if videoCodec != "" {
		mediaType = enums.MediaTypeVideo
	} else if audioCodec != "" {
		mediaType = enums.MediaTypeAudio
	}

	return mediaType, videoCodec, audioCodec
}

func ParseVideoCodec(codecs string) enums.MediaCodec {
	switch {
	case strings.Contains(codecs, "avc"), strings.Contains(codecs, "h264"):
		return enums.MediaCodecAVC
	case strings.Contains(codecs, "hvc"), strings.Contains(codecs, "h265"):
		return enums.MediaCodecHEVC
	case strings.Contains(codecs, "av01"), strings.Contains(codecs, "av1"):
		return enums.MediaCodecAV1
	case strings.Contains(codecs, "vp9"):
		return enums.MediaCodecVP9
	case strings.Contains(codecs, "vp8"):
		return enums.MediaCodecVP8
	default:
		return ""
	}
}

func ParseAudioCodec(codecs string) enums.MediaCodec {
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

func ParseInvURL(url string) string {
	if strings.HasPrefix(url, invInstance) {
		return url
	}
	return invInstance + url
}

func GetInvInstance(cfg *models.ExtractorConfig) (string, error) {
	if invInstance != "" {
		return invInstance, nil
	}
	if cfg.Instance == "" {
		return "", ErrInvidiousNotSet
	}
	parsedURL, err := url.Parse(cfg.Instance)
	if err != nil {
		return "", fmt.Errorf("failed to parse youtube instance url: %w", err)
	}
	invInstance = strings.TrimSuffix(parsedURL.String(), "/")
	return invInstance, nil
}
