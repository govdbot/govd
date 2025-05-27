package models

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/govdbot/govd/enums"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/guregu/null/v6/zero"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	fileTypeDocument = "document"
	fileTypePhoto    = "photo"
	fileTypeVideo    = "video"
	fileTypeAudio    = "audio"

	fileExtMP4  = "mp4"
	fileExtWebM = "webm"
	fileExtMP3  = "mp3"
	fileExtM4A  = "m4a"
	fileExtFLAC = "flac"
	fileExtOGG  = "oga"
	fileExtJPEG = "jpeg"
	fileExtWebP = "webp"
	fileExtJPG  = "jpg"
	fileExtGIF  = "gif"
	fileExtOGV  = "ogv"
	fileExtAVI  = "avi"
	fileExtMKV  = "mkv"
	fileExtMOV  = "mov"
)

type Media struct {
	ID                uint           `json:"-"`
	ContentID         string         `gorm:"not null;index" json:"content_id"`
	ContentURL        string         `gorm:"not null" json:"content_url"`
	ExtractorCodeName string         `gorm:"not null;index" json:"extractor_code_name"`
	Caption           zero.String    `json:"caption"`
	NSFW              bool           `gorm:"default:false" json:"nsfw"`
	CreatedAt         time.Time      `json:"-"`
	UpdatedAt         time.Time      `json:"-"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	Format *MediaFormat `json:"-"`

	Formats []*MediaFormat `gorm:"-" json:"formats"`
}

type MediaFormat struct {
	ID         uint             `json:"-"`
	MediaID    uint             `gorm:"index:idx_media_format,priority:1;not null" json:"-"`
	Type       enums.MediaType  `gorm:"not null;index:idx_media_type" json:"type"`
	FormatID   string           `gorm:"not null;index" json:"format_id"`
	FileID     string           `gorm:"not null;index" json:"-"`
	VideoCodec enums.MediaCodec `json:"video_codec"`
	AudioCodec enums.MediaCodec `json:"audio_codec"`
	Duration   int64            `json:"duration"`
	Width      int64            `json:"width"`
	Height     int64            `json:"height"`
	Bitrate    int64            `json:"bitrate"`
	Title      string           `json:"title"`
	Artist     string           `json:"artist"`
	IsDefault  bool             `gorm:"default:false;index" json:"is_default"`
	Segments   []string         `gorm:"-" json:"segments"`
	FileSize   int64            `json:"-"`
	Plugins    []Plugin         `gorm:"-" json:"-"`

	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// api use only, not stored in database
	URL       []string `gorm:"-" json:"url"`
	Thumbnail []string `gorm:"-" json:"thumbnail"`

	Media *Media `gorm:"foreignKey:MediaID" json:"-"`

	DownloadConfig *DownloadConfig `gorm:"-" json:"-"`
}

type DownloadedMedia struct {
	FilePath          string
	ThumbnailFilePath string
	Media             *Media
	Index             int
}

func (media *Media) GetFormat(formatID string) *MediaFormat {
	for _, format := range media.Formats {
		if format.FormatID == formatID {
			return format
		}
	}
	return nil
}

func (media *Media) GetDefaultFormat() *MediaFormat {
	format := media.GetDefaultVideoFormat()
	if format != nil {
		return format
	}
	format = media.GetDefaultAudioFormat()
	if format != nil {
		return format
	}
	format = media.GetDefaultPhotoFormat()
	if format != nil {
		return format
	}
	return nil
}

func (media *Media) GetDefaultVideoFormat() *MediaFormat {
	filtered := filterFormats(media.Formats, func(format *MediaFormat) bool {
		return format.VideoCodec == enums.MediaCodecAVC
	})
	if len(filtered) == 0 {
		filtered = filterFormats(media.Formats, func(format *MediaFormat) bool {
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
	bestFormat.IsDefault = true
	return bestFormat
}

func (media *Media) GetDefaultAudioFormat() *MediaFormat {
	filtered := filterFormats(media.Formats, func(format *MediaFormat) bool {
		return format.VideoCodec == "" &&
			(format.AudioCodec == enums.MediaCodecAAC ||
				format.AudioCodec == enums.MediaCodecMP3)
	})
	if len(filtered) == 0 {
		filtered = filterFormats(media.Formats, func(format *MediaFormat) bool {
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
	bestFormat.IsDefault = true
	return bestFormat
}

func (media *Media) GetDefaultPhotoFormat() *MediaFormat {
	filtered := filterFormats(media.Formats, func(format *MediaFormat) bool {
		return format.Type == enums.MediaTypePhoto
	})
	if len(filtered) == 0 {
		return nil
	}
	filtered[0].IsDefault = true
	return filtered[0]
}

func (media *Media) GetAudioFromVideoFormat() *MediaFormat {
	videoFormat := media.GetDefaultVideoFormat()

	if videoFormat == nil {
		return nil
	}

	return &MediaFormat{
		Type:       enums.MediaTypeAudio,
		FormatID:   "AudioFromVideo",
		URL:        videoFormat.URL,
		AudioCodec: enums.MediaCodecAAC,
		Thumbnail:  videoFormat.Thumbnail,
		Duration:   videoFormat.Duration,
		Title:      videoFormat.Title,
		Artist:     videoFormat.Artist,
	}
}

func (media *Media) SetCaption(caption string) {
	if len(caption) == 0 {
		return
	}
	media.Caption = zero.StringFrom(caption)
}

func (media *Media) AddFormat(fmt *MediaFormat) {
	media.Formats = append(media.Formats, fmt)
}

func (media *Media) GetSortedFormats() []*MediaFormat {
	// group by video format (codec, width, height)
	groupedVideos := make(map[[3]int64]*MediaFormat)
	for _, format := range media.Formats {
		if format.Type == enums.MediaTypeVideo {
			key := [3]int64{
				getCodecPriority(format.VideoCodec),
				format.Width,
				format.Height,
			}
			existing, ok := groupedVideos[key]
			if !ok || format.Bitrate > existing.Bitrate {
				groupedVideos[key] = format
			}
		}
	}

	// group by audio format (codec, bitrate)
	groupedAudios := make(map[[2]int64]*MediaFormat)
	for _, format := range media.Formats {
		if format.Type == enums.MediaTypeAudio {
			key := [2]int64{
				getCodecPriority(format.AudioCodec),
				format.Bitrate,
			}
			_, exists := groupedAudios[key]
			if !exists {
				groupedAudios[key] = format
			}
		}
	}

	// combine the best video and audio into a final list
	finalSortedList := make([]*MediaFormat, 0, len(groupedVideos)+len(groupedAudios)+len(media.Formats))
	for _, best := range groupedVideos {
		finalSortedList = append(finalSortedList, best)
	}
	for _, best := range groupedAudios {
		finalSortedList = append(finalSortedList, best)
	}

	for _, format := range media.Formats {
		if format.Type != enums.MediaTypeVideo && format.Type != enums.MediaTypeAudio {
			finalSortedList = append(finalSortedList, format) // for non-video and non-audio formats
		}
	}

	// sort the final list
	sort.Slice(finalSortedList, func(i, j int) bool {
		a, b := finalSortedList[i], finalSortedList[j]
		// compare by type priority (video, audio, photo, etc.)
		if cmp := getTypePriority(a.Type) - getTypePriority(b.Type); cmp != 0 {
			return cmp < 0
		}
		// compare by codec priority (for both video and audio)
		if a.Type == enums.MediaTypeVideo {
			if cmp := getCodecPriority(a.VideoCodec) - getCodecPriority(b.VideoCodec); cmp != 0 {
				return cmp < 0
			}
		} else if a.Type == enums.MediaTypeAudio {
			if cmp := getCodecPriority(a.AudioCodec) - getCodecPriority(b.AudioCodec); cmp != 0 {
				return cmp < 0
			}
		}
		// compare by width for videos
		if cmp := a.Width - b.Width; cmp != 0 {
			return cmp < 0
		}
		// compare by height for videos
		if cmp := a.Height - b.Height; cmp != 0 {
			return cmp < 0
		}
		// compare by bitrate (lower bitrate first)
		return a.Bitrate-b.Bitrate < 0
	})

	return finalSortedList
}

func filterFormats(
	formats []*MediaFormat,
	condition func(*MediaFormat) bool,
) []*MediaFormat {
	var filtered []*MediaFormat
	for _, format := range formats {
		if condition(format) {
			filtered = append(filtered, format)
		}
	}
	return filtered
}

func getCodecPriority(codec enums.MediaCodec) int64 {
	codecPriority := map[enums.MediaCodec]int64{
		enums.MediaCodecAVC:  1,
		enums.MediaCodecHEVC: 2,
		enums.MediaCodecMP3:  3,
		enums.MediaCodecAAC:  4,
	}
	return codecPriority[codec]
}

func getTypePriority(mediaType enums.MediaType) int64 {
	typePriority := map[enums.MediaType]int64{
		enums.MediaTypeVideo: 1,
		enums.MediaTypeAudio: 2,
		enums.MediaTypePhoto: 3,
	}
	return typePriority[mediaType]
}

// getFormatInfo returns the file extension and the InputMedia type.
func (format *MediaFormat) GetFormatInfo() (string, string) {
	if format.Type == enums.MediaTypePhoto {
		return fileExtJPEG, fileTypePhoto
	}

	videoCodec := format.VideoCodec
	audioCodec := format.AudioCodec

	switch {
	case videoCodec == enums.MediaCodecAVC && audioCodec == enums.MediaCodecAAC:
		return fileExtMP4, fileTypeVideo
	case videoCodec == enums.MediaCodecAVC && audioCodec == enums.MediaCodecMP3:
		return fileExtMP4, fileTypeVideo
	case videoCodec == enums.MediaCodecHEVC && audioCodec == enums.MediaCodecAAC:
		return fileExtMP4, fileTypeDocument
	case videoCodec == enums.MediaCodecHEVC && audioCodec == enums.MediaCodecMP3:
		return fileExtMP4, fileTypeDocument
	case videoCodec == enums.MediaCodecAVC && audioCodec == "":
		return fileExtMP4, fileTypeVideo
	case videoCodec == enums.MediaCodecHEVC && audioCodec == "":
		return fileExtMP4, fileTypeDocument
	case videoCodec == enums.MediaCodecWebP && audioCodec == "":
		return fileExtWebP, fileTypeVideo
	case videoCodec == "" && audioCodec == enums.MediaCodecMP3:
		return fileExtMP3, fileTypeAudio
	case videoCodec == "" && audioCodec == enums.MediaCodecAAC:
		return fileExtM4A, fileTypeAudio
	case videoCodec == "" && audioCodec == enums.MediaCodecFLAC:
		return fileExtFLAC, fileTypeDocument
	case videoCodec == "" && audioCodec == enums.MediaCodecVorbis:
		return fileExtOGG, fileTypeDocument
	default:
		// all other cases, we return webm as document
		return fileExtWebM, fileTypeDocument
	}
}

func (format *MediaFormat) GetInputMedia(
	filePath string,
	thumbnailFilePath string,
	messageCaption string,
) (gotgbot.InputMedia, error) {
	if format.FileID != "" {
		return format.GetInputMediaWithFileID(messageCaption)
	}

	_, inputMediaType := format.GetFormatInfo()

	fileObj, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	fileInfo, err := fileObj.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	fileSize := fileInfo.Size()

	zap.S().Debugf("file size: %s", humanize.Bytes(uint64(fileSize)))

	fileInputMedia := gotgbot.InputFileByReader(
		filepath.Base(filePath),
		fileObj,
	)

	var thumbnailFileInputMedia gotgbot.InputFile
	if thumbnailFilePath != "" {
		thumbnailFileObj, err := os.Open(thumbnailFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file: %w", err)
		}
		thumbnailFileInputMedia = gotgbot.InputFileByReader(
			filepath.Base(thumbnailFilePath),
			thumbnailFileObj,
		)
	}

	switch inputMediaType {
	case fileTypeVideo:
		return &gotgbot.InputMediaVideo{
			Media:             fileInputMedia,
			Thumbnail:         thumbnailFileInputMedia,
			Width:             format.Width,
			Height:            format.Height,
			Duration:          format.Duration,
			Caption:           messageCaption,
			SupportsStreaming: true,
			ParseMode:         gotgbot.ParseModeHTML,
		}, nil
	case fileTypeAudio:
		return &gotgbot.InputMediaAudio{
			Media:     fileInputMedia,
			Thumbnail: thumbnailFileInputMedia,
			Duration:  format.Duration,
			Performer: format.Artist,
			Title:     format.Title,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	case fileTypePhoto:
		return &gotgbot.InputMediaPhoto{
			Media:     fileInputMedia,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	case fileTypeDocument:
		return &gotgbot.InputMediaDocument{
			Media:     fileInputMedia,
			Thumbnail: thumbnailFileInputMedia,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	default:
		return nil, fmt.Errorf("unknown input type: %s", inputMediaType)
	}
}

func (format *MediaFormat) GetInputMediaWithFileID(
	messageCaption string,
) (gotgbot.InputMedia, error) {
	_, inputMediaType := format.GetFormatInfo()
	fileInputMedia := gotgbot.InputFileByID(format.FileID)
	switch inputMediaType {
	case fileTypeVideo:
		return &gotgbot.InputMediaVideo{
			Media:     fileInputMedia,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	case fileTypeAudio:
		return &gotgbot.InputMediaAudio{
			Media:     fileInputMedia,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	case fileTypePhoto:
		return &gotgbot.InputMediaPhoto{
			Media:     fileInputMedia,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	case fileTypeDocument:
		return &gotgbot.InputMediaDocument{
			Media:     fileInputMedia,
			Caption:   messageCaption,
			ParseMode: gotgbot.ParseModeHTML,
		}, nil
	default:
		return nil, fmt.Errorf("unknown input type: %s", inputMediaType)
	}
}

func (format *MediaFormat) GetFileName() string {
	extension, _ := format.GetFormatInfo()
	if format.Type == enums.MediaTypeAudio && format.Title != "" && format.Artist != "" {
		artist := strings.ReplaceAll(format.Artist, "/", " ")
		title := strings.ReplaceAll(format.Title, "/", " ")
		return fmt.Sprintf("%s - %s.%s", artist, title, extension)
	}
	name := uuid.New().String()
	name = strings.ReplaceAll(name, "-", "")
	return fmt.Sprintf("%s.%s", name, extension)
}

func (media *Media) HasVideo() bool {
	for _, format := range media.Formats {
		if format.Type == enums.MediaTypeVideo {
			return true
		}
	}
	return false
}

func (media *Media) HasAudio() bool {
	for _, format := range media.Formats {
		if format.Type == enums.MediaTypeAudio {
			return true
		}
	}
	return false
}

func (media *Media) HasPhoto() bool {
	for _, format := range media.Formats {
		if format.Type == enums.MediaTypePhoto {
			return true
		}
	}
	return false
}

func (media *Media) SupportsAudio() bool {
	for _, format := range media.Formats {
		if format.AudioCodec != "" {
			return true
		}
	}
	return false
}

func (media *Media) SupportsAudioFromVideo() bool {
	return !media.HasAudio() && media.HasVideo() && media.SupportsAudio()
}
