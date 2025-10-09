package models

import (
	"fmt"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/govdbot/govd/internal/database"
)

type Media struct {
	ContentID   string
	ContentURL  string
	ExtractorID string
	Caption     string
	NSFW        bool

	Items []*MediaItem
}

func (m *Media) NewItem() *MediaItem {
	item := &MediaItem{
		Formats: make([]*MediaFormat, 0),
	}
	m.Items = append(m.Items, item)
	return item
}

func (m *Media) AddItems(items ...*MediaItem) {
	m.Items = append(m.Items, items...)
}

func (m *Media) SetCaption(caption string) {
	if m.Caption != "" {
		return
	}
	m.Caption = caption
}

func (m *Media) SetNSFW() {
	m.NSFW = true
}

type MediaItem struct {
	Formats []*MediaFormat
}

type MediaFormat struct {
	FormatID   string
	FileID     string
	Type       database.MediaType
	AudioCodec database.MediaCodec
	VideoCodec database.MediaCodec
	Duration   int32
	Title      string
	Artist     string
	Width      int32
	Height     int32
	Bitrate    int32

	URL          []string
	ThumbnailURL []string
}

func NewMediaFromDB(
	media *database.Media,
) *Media {
	// TODO
	return nil
}

// returns the file extension and the InputMedia type.
func (f *MediaFormat) GetInfo() (FileExtension, FileType) {
	if f.Type == database.MediaTypePhoto {
		return FileExtensionJPEG, FileTypePhoto
	}

	videoCodec := f.VideoCodec
	audioCodec := f.AudioCodec

	switch {
	case videoCodec == database.MediaCodecAvc && audioCodec == database.MediaCodecAac:
		return FileExtensionMP4, FileTypeVideo
	case videoCodec == database.MediaCodecAvc && audioCodec == database.MediaCodecMp3:
		return FileExtensionMP4, FileTypeVideo
	case videoCodec == database.MediaCodecHevc && audioCodec == database.MediaCodecAac:
		return FileExtensionMP4, FileTypeDocument
	case videoCodec == database.MediaCodecHevc && audioCodec == database.MediaCodecMp3:
		return FileExtensionMP4, FileTypeDocument
	case videoCodec == database.MediaCodecAvc && audioCodec == "":
		return FileExtensionMP4, FileTypeVideo
	case videoCodec == database.MediaCodecHevc && audioCodec == "":
		return FileExtensionMP4, FileTypeDocument
	case videoCodec == database.MediaCodecWebp && audioCodec == "":
		return FileExtensionWEBP, FileTypeVideo
	case videoCodec == "" && audioCodec == database.MediaCodecMp3:
		return FileExtensionMP3, FileTypeAudio
	case videoCodec == "" && audioCodec == database.MediaCodecAac:
		return FileExtensionM4A, FileTypeAudio
	case videoCodec == "" && audioCodec == database.MediaCodecFlac:
		return FileExtensionFLAC, FileTypeDocument
	case videoCodec == "" && audioCodec == database.MediaCodecVorbis:
		return FileExtensionOGG, FileTypeDocument
	default:
		// all other cases, we return webm as document
		return FileExtensionWEBM, FileTypeDocument
	}
}

func (f *MediaFormat) GetFileName() string {
	ext, _ := f.GetInfo()
	if f.Type == database.MediaTypeAudio && f.Title != "" && f.Artist != "" {
		artist := strings.ReplaceAll(f.Artist, "/", " ")
		title := strings.ReplaceAll(f.Title, "/", " ")
		return fmt.Sprintf("%s - %s.%s", artist, title, ext)
	}
	name := uuid.New().String()
	name = strings.ReplaceAll(name, "-", "")
	return fmt.Sprintf("%s.%s", name, ext)
}

func (mi *MediaItem) AddFormats(formats ...*MediaFormat) {
	mi.Formats = append(mi.Formats, formats...)
}

func (mi *MediaItem) GetFormatByID(formatID string) *MediaFormat {
	for _, format := range mi.Formats {
		if format.FormatID == formatID {
			return format
		}
	}
	return nil
}

func (mi *MediaItem) GetDefaultFormat() *MediaFormat {
	format := mi.GetDefaultVideoFormat()
	if format != nil {
		return format
	}
	format = mi.GetDefaultAudioFormat()
	if format != nil {
		return format
	}
	format = mi.GetDefaultPhotoFormat()
	if format != nil {
		return format
	}
	return nil
}

func (mi *MediaItem) GetDefaultVideoFormat() *MediaFormat {
	filtered := mi.FilterFormats(func(format *MediaFormat) bool {
		return format.VideoCodec == database.MediaCodecAvc
	})
	if len(filtered) == 0 {
		filtered = mi.FilterFormats(func(format *MediaFormat) bool {
			return format.VideoCodec != ""
		})
	}
	if len(filtered) == 0 {
		return nil
	}
	slices.SortFunc(filtered, func(a, b *MediaFormat) int {
		if a.Bitrate != b.Bitrate {
			if a.Bitrate > b.Bitrate {
				return -1
			}
			return 1
		}
		if a.Height > b.Height {
			return -1
		} else if a.Height < b.Height {
			return 1
		}
		return 0
	})
	bestFormat := filtered[0]
	return bestFormat
}

func (mi *MediaItem) GetDefaultAudioFormat() *MediaFormat {
	filtered := mi.FilterFormats(func(format *MediaFormat) bool {
		return format.VideoCodec == "" &&
			(format.AudioCodec == database.MediaCodecAac ||
				format.AudioCodec == database.MediaCodecMp3)
	})
	if len(filtered) == 0 {
		filtered = mi.FilterFormats(func(format *MediaFormat) bool {
			return format.VideoCodec == "" && format.AudioCodec != ""
		})
	}
	if len(filtered) == 0 {
		return nil
	}
	bestFormat := filtered[0]
	for _, format := range filtered {
		if format.Bitrate > bestFormat.Bitrate {
			bestFormat = format
		}
	}
	return bestFormat
}

func (mi *MediaItem) GetDefaultPhotoFormat() *MediaFormat {
	filtered := mi.FilterFormats(func(format *MediaFormat) bool {
		return format.Type == database.MediaTypePhoto
	})
	if len(filtered) == 0 {
		return nil
	}
	return filtered[0]
}

func (mi *MediaItem) FilterFormats(
	condition func(*MediaFormat) bool,
) []*MediaFormat {
	filtered := make([]*MediaFormat, 0, len(mi.Formats))
	for _, format := range mi.Formats {
		if condition(format) {
			filtered = append(filtered, format)
		}
	}
	return filtered
}
