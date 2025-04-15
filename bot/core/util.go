package core

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	"govd/database"
	"govd/enums"
	"govd/models"
	"govd/plugins"
	"govd/util"
	"govd/util/av"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

func getFileThumbnail(
	format *models.MediaFormat,
	filePath string,
) (string, error) {
	fileDir := filepath.Dir(filePath)
	fileName := filepath.Base(filePath)
	fileExt := filepath.Ext(fileName)
	fileBaseName := fileName[:len(fileName)-len(fileExt)]
	thumbnailFilePath := filepath.Join(fileDir, fileBaseName+".thumb.jpeg")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if len(format.Thumbnail) > 0 {
		file, err := util.DownloadFileInMemory(ctx, format.Thumbnail, nil)
		if err != nil {
			return "", fmt.Errorf("failed to download file in memory: %w", err)
		}
		err = util.ImgToJPEG(file, thumbnailFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to convert to JPEG: %w", err)
		}
		return thumbnailFilePath, nil
	}
	if format.Type == enums.MediaTypeVideo {
		err := av.ExtractVideoThumbnail(filePath, thumbnailFilePath)
		if err != nil {
			return "", fmt.Errorf("failed to extract video thumbnail: %w", err)
		}
		return thumbnailFilePath, nil
	}
	return "", nil
}

func insertVideoInfo(
	format *models.MediaFormat,
	filePath string,
) {
	duration, width, height := av.GetVideoInfo(filePath)
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
	var storedMedias []*models.Media
	if len(medias) == 0 {
		return fmt.Errorf("no media to store")
	}
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
			return fmt.Errorf("failed to store media: %w", err)
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
	ctx *ext.Context,
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
	ctx *ext.Context,
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
	currentError := err

	if errors.Is(currentError, context.Canceled) ||
		errors.Is(currentError, context.DeadlineExceeded) {
		SendErrorMessage(
			bot, ctx,
			"download request canceled or timed out",
		)
		return
	}

	for currentError != nil {
		var botError *util.Error
		if errors.As(currentError, &botError) {
			SendErrorMessage(bot, ctx, fmt.Sprintf(
				"error occurred when downloading: %s",
				currentError.Error(),
			))
			return
		}
		currentError = errors.Unwrap(currentError)
	}

	lastError := util.GetLastError(err)
	errorMessage := fmt.Sprintf(
		"error occurred when downloading: %s",
		lastError.Error(),
	)

	if strings.Contains(errorMessage, bot.Token) {
		errorMessage = "telegram related error, probably connection issue"
	}

	SendErrorMessage(bot, ctx, errorMessage)

}

func SendErrorMessage(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	errorMessage string,
) {
	switch {
	case ctx.Update.Message != nil:
		ctx.EffectiveMessage.Reply(
			bot,
			errorMessage,
			nil,
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
			})
	}
}

func ensureMergeFormats(
	media *models.Media,
	videoFormat *models.MediaFormat,
) {
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
