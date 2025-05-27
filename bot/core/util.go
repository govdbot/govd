package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/govdbot/govd/database"
	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/plugins"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/libav"
	"github.com/govdbot/govd/util/mp4box"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func ValidateMediaList(
	mediaList []*models.Media,
) error {
	for i := range mediaList {
		defaultFormat := mediaList[i].GetDefaultFormat()
		if defaultFormat == nil {
			return fmt.Errorf("no default format found for media at index %d", i)
		}
		if len(defaultFormat.URL) == 0 {
			return fmt.Errorf("media format at index %d has no URL", i)
		}

		zap.S().Debugf("default format selected: %s (media %d)", defaultFormat.FormatID, i)

		// ensure we can merge video and audio formats
		EnsureMergeFormats(mediaList[i], defaultFormat)

		// ensure download config is set
		if defaultFormat.DownloadConfig == nil {
			defaultFormat.DownloadConfig = models.GetDownloadConfig(nil)
		}

		// check for file size and duration limits
		if util.ExceedsMaxFileSize(defaultFormat.FileSize) {
			return util.ErrFileTooLarge
		}
		if util.ExceedsMaxDuration(defaultFormat.Duration) {
			return util.ErrDurationTooLong
		}

		mediaList[i].Format = defaultFormat
	}
	return nil
}

func GetFileThumbnail(
	ctx context.Context,
	format *models.MediaFormat,
	filePath string,
	downloadConfig *models.DownloadConfig,
) (string, error) {
	fileDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(fileName)
	fileBaseName := fileName[:len(fileName)-len(fileExt)]
	thumbnailFilePath := filepath.Join(fileDir, fileBaseName+".thumb.jpeg")

	if len(format.Thumbnail) > 0 {
		zap.S().Debug("downloading thumbnail from URL")
		file, err := util.DownloadFileInMemory(ctx, format.Thumbnail, downloadConfig)
		if err != nil {
			return "", fmt.Errorf("failed to download file in memory: %w", err)
		}
		if format.Type == enums.MediaTypeAudio {
			zap.S().Debug("resizing audio thumbnail")
			// thumbnails for audio files are usually 320x320
			err = util.ResizeImgToJPEG(file, thumbnailFilePath, 320)
			if err != nil {
				return "", fmt.Errorf("failed to convert to JPEG: %w", err)
			}
		} else {
			err = util.ImgToJPEG(file, thumbnailFilePath)
			if err != nil {
				return "", fmt.Errorf("failed to convert to JPEG: %w", err)
			}
		}
		return thumbnailFilePath, nil
	}
	if format.Type == enums.MediaTypeVideo {
		zap.S().Debug("extracting video thumbnail with libav")
		err := libav.ExtractVideoThumbnail(filePath, thumbnailFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to extract video thumbnail: %w", err)
		}
		return thumbnailFilePath, nil
	}
	return "", nil
}

func InsertVideoInfo(
	format *models.MediaFormat,
	filePath string,
) {
	zap.S().Debug("extracting video info from mp4box")
	duration, width, height := mp4box.ExtractBoxMetadata(filePath)
	if duration == 0 && width == 0 && height == 0 {
		zap.S().Debug("extracting video info with libav")
		duration, width, height = libav.GetVideoInfo(filePath)
	}
	format.Duration = duration
	format.Width = width
	format.Height = height
}

func GetMessageFileID(msg *gotgbot.Message) string {
	switch {
	case msg.Video != nil:
		return msg.Video.FileId
	case msg.Animation != nil:
		return msg.Animation.FileId
	case msg.Photo != nil:
		return msg.Photo[len(msg.Photo)-1].FileId
	case msg.Document != nil:
		return msg.Document.FileId
	case msg.Audio != nil:
		return msg.Audio.FileId
	case msg.Voice != nil:
		return msg.Voice.FileId
	default:
		return ""
	}
}

func GetMessageFileSize(msg *gotgbot.Message) int64 {
	switch {
	case msg.Video != nil:
		return msg.Video.FileSize
	case msg.Animation != nil:
		return msg.Animation.FileSize
	case msg.Photo != nil:
		return msg.Photo[len(msg.Photo)-1].FileSize
	case msg.Document != nil:
		return msg.Document.FileSize
	case msg.Audio != nil:
		return msg.Audio.FileSize
	case msg.Voice != nil:
		return msg.Voice.FileSize
	default:
		return 0
	}
}

func StoreMedias(
	dlCtx *models.DownloadContext,
	msgs []gotgbot.Message,
	medias []*models.DownloadedMedia,
) error {
	if len(medias) == 0 {
		return errors.New("no media to store")
	}

	zap.S().Debugf(
		"storing %d medias for %s (%s)",
		len(medias),
		dlCtx.MatchedContentID,
		dlCtx.Extractor.CodeName,
	)

	storedMedias := make([]*models.Media, 0, len(medias))

	for idx, msg := range msgs {
		fileID := GetMessageFileID(&msg)
		if len(fileID) == 0 {
			return fmt.Errorf("no file ID found for media at index %d", idx)
		}
		fileSize := GetMessageFileSize(&msg)
		medias[idx].Media.Format.FileID = fileID
		medias[idx].Media.Format.FileSize = fileSize
		storedMedias = append(
			storedMedias,
			medias[idx].Media,
		)
	}
	for _, media := range storedMedias {
		err := database.StoreMedia(
			dlCtx.Extractor.CodeName,
			media.ContentID,
			media,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func FormatCaption(
	media *models.Media,
	isEnabled bool,
) string {
	newCaption := fmt.Sprintf(
		"<a href='%s'>source</a> - @govd_bot\n",
		media.ContentURL,
	)
	if isEnabled && media.Caption.Valid {
		text := media.Caption.String
		if len(text) > 600 {
			text = text[:600] + "..."
		}
		newCaption += fmt.Sprintf(
			"<blockquote expandable>%s</blockquote>\n",
			util.EscapeCaption(text),
		)
	}
	return newCaption
}

func TypingEffect(
	bot *gotgbot.Bot,
	chatID int64,
) {
	bot.SendChatAction(
		chatID,
		"typing",
		nil,
	)
}

func SendingEffect(
	bot *gotgbot.Bot,
	chatID int64,
	mediaType enums.MediaType,
) {
	action := "upload_document"
	if mediaType == enums.MediaTypeVideo {
		action = "upload_video"
	}
	if mediaType == enums.MediaTypeAudio {
		action = "upload_audio"
	}
	if mediaType == enums.MediaTypePhoto {
		action = "upload_photo"
	}
	bot.SendChatAction(
		chatID,
		action,
		nil,
	)
}

func HandleErrorMessage(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	err error,
) {
	if zap.S().Level() == zap.DebugLevel {
		zap.S().Errorf("error occurred: %v", err)
	}

	if strings.Contains(err.Error(), bot.Token) {
		errorMessage := "telegram related error, probably connection issues"
		SendErrorMessage(bot, ctx, errorMessage)
		return
	}

	var telegramError *gotgbot.TelegramError
	if errors.As(err, &telegramError) {
		if zap.S().Level() == zap.DebugLevel {
			zap.S().Errorf(
				"telegram error: Code=%d, Description=%s",
				telegramError.Code,
				telegramError.Description,
			)
		}
		// check if error is related to botapi file size limit
		if telegramError.Description == "Request Entity Too Large" {
			err = util.ErrTelegramFileTooLarge
		}
	}

	if errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) {
		errorMessage := "download request canceled or timed out"
		SendErrorMessage(bot, ctx, errorMessage)
		return
	}

	currentError := err
	for currentError != nil {
		var botError *util.Error
		if errors.As(currentError, &botError) {
			errorMessage := "error occurred when downloading: " + currentError.Error()
			SendErrorMessage(bot, ctx, errorMessage)
			return
		}
		currentError = errors.Unwrap(currentError)
	}
	errorMessage := "error occurred when downloading: " + err.Error()
	SendErrorMessage(bot, ctx, errorMessage)
}

func SendErrorMessage(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	errorMessage string,
) {
	// avoid leaking sensitive URLs in error messages
	errorMessage = util.RedactURLs(errorMessage)

	switch {
	case ctx.Update.Message != nil:
		ctx.EffectiveMessage.Reply(
			bot,
			errorMessage,
			&gotgbot.SendMessageOpts{
				LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
					IsDisabled: true,
				},
			},
		)
	case ctx.Update.InlineQuery != nil:
		ctx.InlineQuery.Answer(
			bot,
			nil,
			&gotgbot.AnswerInlineQueryOpts{
				CacheTime: 1,
				Button: &gotgbot.InlineQueryResultsButton{
					Text:           errorMessage,
					StartParameter: "start",
				},
			},
		)
	case ctx.ChosenInlineResult != nil:
		bot.EditMessageText(
			errorMessage,
			&gotgbot.EditMessageTextOpts{
				InlineMessageId: ctx.ChosenInlineResult.InlineMessageId,
				LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
					IsDisabled: true,
				},
			},
		)
	}
}

func EnsureMergeFormats(
	media *models.Media,
	videoFormat *models.MediaFormat,
) {
	zap.S().Debugf(
		"ensuring merge formats for %s (%s)",
		media.ContentID, media.ExtractorCodeName,
	)
	if videoFormat.Type != enums.MediaTypeVideo {
		return
	}
	if videoFormat.AudioCodec != "" {
		return
	}
	// video with no audio
	audioFormat := media.GetDefaultAudioFormat()
	if audioFormat == nil {
		return
	}
	videoFormat.AudioCodec = audioFormat.AudioCodec
	videoFormat.Plugins = append(videoFormat.Plugins, plugins.MergeAudio)
}
