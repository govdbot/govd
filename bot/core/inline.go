package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/govdbot/govd/database"
	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

type TaskEntry struct {
	Task      *models.DownloadContext
	CreatedAt time.Time
}

var InlineTasks sync.Map
var cleanupActive sync.Once

const taskTimeout = 5 * time.Minute

func GetTask(id string) (*models.DownloadContext, error) {
	value, ok := InlineTasks.Load(id)
	if !ok {
		return nil, errors.New("task not found")
	}
	entry, ok := value.(TaskEntry)
	if !ok {
		return nil, errors.New("invalid task entry")
	}
	zap.S().Debugf("inline task %s found", id)
	return entry.Task, nil
}

func SetTask(id string, task *models.DownloadContext) {
	zap.S().Debugf("setting inline task %s", id)
	InlineTasks.Store(id, TaskEntry{
		Task:      task,
		CreatedAt: time.Now(),
	})
	cleanupActive.Do(StartTasksCleanup)
}

func DeleteTask(id string) {
	zap.S().Debugf("deleting inline task %s", id)
	InlineTasks.Delete(id)
}

func StartTasksCleanup() {
	zap.S().Debug("starting inline tasks cleanup")
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CleanupStaleTasks()
		}
	}()
}

func CleanupStaleTasks() {
	now := time.Now()
	InlineTasks.Range(func(key, value any) bool {
		entry, ok := value.(TaskEntry)
		if !ok {
			InlineTasks.Delete(key)
			return true
		}

		if now.Sub(entry.CreatedAt) > taskTimeout {
			zap.S().Debugf("deleting stale task %s", key)
			InlineTasks.Delete(key)
		}
		return true
	})
}

func HandleInline(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	dlCtx *models.DownloadContext,
) error {
	if dlCtx.Extractor.Type != enums.ExtractorTypeSingle {
		return util.ErrNotImplemented
	}
	contentID := dlCtx.MatchedContentID
	cached, err := database.GetDefaultMedias(
		dlCtx.Extractor.CodeName,
		contentID,
	)
	if err != nil {
		return err
	}
	if len(cached) > 0 {
		if len(cached) > 1 {
			return util.ErrInlineMediaGroup
		}
		err = HandleInlineCached(
			bot, ctx, cached[0],
		)
		if err != nil {
			return err
		}
		return nil
	}
	err = StartInlineTask(bot, ctx, dlCtx)
	if err != nil {
		return err
	}
	return nil
}

func HandleInlineCached(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	media *models.Media,
) error {
	var result gotgbot.InlineQueryResult

	format := media.Format
	resultID := fmt.Sprintf("%d:%s", ctx.EffectiveUser.Id, format.FormatID)
	resultTitle := "share"
	mediaCaption := FormatCaption(media, true)
	_, inputFileType := format.GetFormatInfo()

	switch inputFileType {
	case "photo":
		result = &gotgbot.InlineQueryResultCachedPhoto{
			Id:          resultID,
			PhotoFileId: format.FileID,
			Title:       resultTitle,
			Caption:     mediaCaption,
			ParseMode:   gotgbot.ParseModeHTML,
		}
	case "video":
		result = &gotgbot.InlineQueryResultCachedVideo{
			Id:          resultID,
			VideoFileId: format.FileID,
			Title:       resultTitle,
			Caption:     mediaCaption,
			ParseMode:   gotgbot.ParseModeHTML,
		}
	case "audio":
		result = &gotgbot.InlineQueryResultCachedAudio{
			Id:          resultID,
			AudioFileId: format.FileID,
			Caption:     mediaCaption,
			ParseMode:   gotgbot.ParseModeHTML,
		}
	case "document":
		result = &gotgbot.InlineQueryResultCachedDocument{
			Id:             resultID,
			DocumentFileId: format.FileID,
			Title:          resultTitle,
			Caption:        mediaCaption,
			ParseMode:      gotgbot.ParseModeHTML,
		}
	default:
		return errors.New("unsupported input file type")
	}
	ctx.InlineQuery.Answer(
		bot, []gotgbot.InlineQueryResult{result},
		&gotgbot.AnswerInlineQueryOpts{
			CacheTime:  1,
			IsPersonal: true,
		},
	)
	return nil
}

func HandleInlineCachedResult(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	media *models.Media,
) error {
	format := media.Format
	messageCaption := FormatCaption(media, true)
	inputMedia, err := format.GetInputMediaWithFileID(messageCaption)
	if err != nil {
		return err
	}

	_, _, err = bot.EditMessageMedia(
		inputMedia,
		&gotgbot.EditMessageMediaOpts{
			InlineMessageId: ctx.ChosenInlineResult.InlineMessageId,
		},
	)
	if err != nil {
		return err
	}
	return nil
}

func StartInlineTask(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	dlCtx *models.DownloadContext,
) error {
	randomID, err := uuid.NewUUID()
	if err != nil {
		return errors.New("could not generate task ID")
	}
	taskID := randomID.String()
	inlineResult := &gotgbot.InlineQueryResultArticle{
		Id:    taskID,
		Title: "share",
		InputMessageContent: &gotgbot.InputTextMessageContent{
			MessageText: "loading media please wait...",
			ParseMode:   gotgbot.ParseModeHTML,
			LinkPreviewOptions: &gotgbot.LinkPreviewOptions{
				IsDisabled: true,
			},
		},
		ReplyMarkup: &gotgbot.InlineKeyboardMarkup{
			InlineKeyboard: [][]gotgbot.InlineKeyboardButton{
				{
					{
						Text:         "...",
						CallbackData: "inline:loading",
					},
				},
			},
		},
	}
	ok, err := ctx.InlineQuery.Answer(
		bot, []gotgbot.InlineQueryResult{inlineResult},
		&gotgbot.AnswerInlineQueryOpts{
			CacheTime:  1,
			IsPersonal: true,
		},
	)
	if err != nil || !ok {
		return err
	}
	SetTask(taskID, dlCtx)
	return nil
}

func GetInlineFormat(
	taskCtx context.Context,
	bot *gotgbot.Bot,
	ctx *ext.Context,
	dlCtx *models.DownloadContext,
	mediaChan chan<- *models.Media,
	errChan chan<- error,
) {
	response, err := dlCtx.Extractor.Run(dlCtx)
	if err != nil {
		errChan <- fmt.Errorf("failed to get media: %w", err)
		return
	}
	mediaList := response.MediaList
	if len(mediaList) == 0 {
		errChan <- fmt.Errorf("no media found for content ID: %s", dlCtx.MatchedContentID)
		return
	}
	if len(mediaList) > 1 {
		errChan <- util.ErrInlineMediaGroup
		return
	}

	err = ValidateMediaList(mediaList)
	if err != nil {
		errChan <- err
		return
	}

	messageCaption := FormatCaption(mediaList[0], true)

	medias, err := DownloadMedias(taskCtx, mediaList)
	if err != nil {
		errChan <- fmt.Errorf("failed to download medias: %w", err)
		return
	}

	// plugins act as post-processing for the media.
	// they are run after the media is downloaded
	// and before it is sent to the user
	// this allows for things like merging audio and video, etc.
	for _, media := range medias {
		format := media.Media.Format
		zap.S().Debugf(
			"running %d plugins for %s (%s)",
			len(format.Plugins),
			dlCtx.MatchedContentID,
			dlCtx.Extractor.CodeName,
		)
		for _, plugin := range format.Plugins {
			err = plugin(media, format.DownloadConfig)
			if err != nil {
				errChan <- fmt.Errorf("failed to run plugin: %w", err)
				return
			}
		}
	}

	msgs, err := SendMedias(
		bot, ctx, dlCtx,
		medias, &models.SendMediaFormatsOptions{
			Caption: messageCaption,
		},
	)
	if err != nil {
		errChan <- fmt.Errorf("failed to send media: %w", err)
		return
	}
	msg := &msgs[0]
	msg.Delete(bot, nil)
	mediaChan <- medias[0].Media
}
