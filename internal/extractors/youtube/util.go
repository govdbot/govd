package youtube

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

const invEndpoint = "/api/v1/videos/"

func ParseInvFormats(data *InvResponse, instance string) []*models.MediaFormat {
	formats := make([]*models.MediaFormat, 0, len(data.AdaptiveFormats))
	duration := data.LengthSeconds

	for _, format := range data.AdaptiveFormats {
		if format.URL == "" {
			continue
		}
		if strings.Contains(format.URL, "dubbed-auto") {
			// skip auto-dubbed audio formats
			continue
		}
		mediaType, vCodec, aCodec, err := ParseStreamType(format.Type)
		if err != nil {
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
			URL:        []string{ParseInvURL(format.URL, instance)},
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

func ParseStreamType(streamType string) (database.MediaType, database.MediaCodec, database.MediaCodec, error) {
	parts := strings.Split(streamType, "; ")
	if len(parts) != 2 {
		// unknown stream type
		return "", "", "", fmt.Errorf("unknown stream type: %s", streamType)
	}
	codecs := parts[1]

	var mediaType database.MediaType
	var videoCodec, audioCodec database.MediaCodec

	videoCodec = util.ParseVideoCodec(codecs)
	audioCodec = util.ParseAudioCodec(codecs)

	if videoCodec != "" {
		mediaType = database.MediaTypeVideo
	} else if audioCodec != "" {
		mediaType = database.MediaTypeAudio
	} else {
		// unknown codec
		return "", "", "", fmt.Errorf("unknown codec in stream type: %s", streamType)
	}

	return mediaType, videoCodec, audioCodec, nil
}

func ParseInvURL(url string, instance string) string {
	if strings.HasPrefix(url, instance) {
		return url
	}
	return instance + url
}

func GetInvInstance(ctx *models.ExtractorContext, idx int) (string, error) {
	instance := ctx.Config.Instance[idx]
	if instance == "" {
		return "", fmt.Errorf("no youtube instance configured")
	}
	parsedURL, err := url.Parse(instance)
	if err != nil {
		return "", fmt.Errorf("failed to parse youtube instance url: %w", err)
	}
	return strings.TrimSuffix(parsedURL.String(), "/"), nil
}
