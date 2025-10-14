package youtube

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
)

const invEndpoint = "/api/v1/videos/"

var invInstance string

func ParseInvFormats(data *InvResponse) []*models.MediaFormat {
	formats := make([]*models.MediaFormat, 0, len(data.AdaptiveFormats))
	duration := data.LengthSeconds

	for _, format := range data.AdaptiveFormats {
		if format.URL == "" {
			continue
		}
		mediaType, vCodec, aCodec := ParseStreamType(format.Type)
		if mediaType == "" {
			continue
		}
		var bitrate int64
		if format.Bitrate != "" {
			bitrate, _ = strconv.ParseInt(format.Bitrate, 10, 32)
		}
		var width, height int64
		if format.Size != "" {
			dimensions := strings.Split(format.Size, "x")
			if len(dimensions) == 2 {
				width, _ = strconv.ParseInt(dimensions[0], 10, 32)
				height, _ = strconv.ParseInt(dimensions[1], 10, 32)
			}
		}
		// we dont use thumbnails provided by youtube
		// due to black bars on the sides for some videos
		formats = append(formats, &models.MediaFormat{
			Type:       mediaType,
			VideoCodec: vCodec,
			AudioCodec: aCodec,
			FormatID:   format.Itag,
			Width:      int32(width),
			Height:     int32(height),
			Bitrate:    int32(bitrate),
			Duration:   duration,
			URL:        []string{ParseInvURL(format.URL)},
			Title:      data.Title,
			Artist:     data.Author,
			DownloadSettings: &models.DownloadSettings{
				// youtube throttles the download speed
				// if chunk size is too small
				ChunkSize: 10 * 1024 * 1024, // 10 MB
			},
		})
	}
	return formats
}

func ParseStreamType(streamType string) (database.MediaType, database.MediaCodec, database.MediaCodec) {
	parts := strings.Split(streamType, "; ")
	if len(parts) != 2 {
		// unknown stream type
		return "", "", ""
	}
	codecs := parts[1]

	var mediaType database.MediaType
	var videoCodec, audioCodec database.MediaCodec

	videoCodec = ParseVideoCodec(codecs)
	audioCodec = ParseAudioCodec(codecs)

	if videoCodec != "" {
		mediaType = database.MediaTypeVideo
	} else if audioCodec != "" {
		mediaType = database.MediaTypeAudio
	}

	return mediaType, videoCodec, audioCodec
}

func ParseVideoCodec(codecs string) database.MediaCodec {
	switch {
	case strings.Contains(codecs, "avc"), strings.Contains(codecs, "h264"):
		return database.MediaCodecAvc
	case strings.Contains(codecs, "hvc"), strings.Contains(codecs, "h265"):
		return database.MediaCodecHevc
	case strings.Contains(codecs, "av01"), strings.Contains(codecs, "av1"):
		return database.MediaCodecAv1
	case strings.Contains(codecs, "vp9"):
		return database.MediaCodecVp9
	case strings.Contains(codecs, "vp8"):
		return database.MediaCodecVp8
	default:
		return ""
	}
}

func ParseAudioCodec(codecs string) database.MediaCodec {
	switch {
	case strings.Contains(codecs, "mp4a"):
		return database.MediaCodecAac
	case strings.Contains(codecs, "opus"):
		return database.MediaCodecOpus
	case strings.Contains(codecs, "mp3"):
		return database.MediaCodecMp3
	case strings.Contains(codecs, "flac"):
		return database.MediaCodecFlac
	case strings.Contains(codecs, "vorbis"):
		return database.MediaCodecVorbis
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

func GetInvInstance(ctx *models.ExtractorContext) (string, error) {
	if invInstance != "" {
		return invInstance, nil
	}
	instance := ctx.Config.Instance
	if instance == "" {
		return "", fmt.Errorf("no youtube instance configured")
	}
	parsedURL, err := url.Parse(instance)
	if err != nil {
		return "", fmt.Errorf("failed to parse youtube instance url: %w", err)
	}
	invInstance = strings.TrimSuffix(parsedURL.String(), "/")
	return invInstance, nil
}
