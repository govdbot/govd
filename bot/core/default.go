package core

import (
	"context"
	"fmt"
	"govd/database"
	"govd/models"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"github.com/pkg/errors"
)

func HandleDefaultFormatDownload(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	taskCtx context.Context,
	dlCtx *models.DownloadContext,
) error {
	storedMedias, err := database.GetDefaultMedias(
		dlCtx.Extractor.CodeName,
		dlCtx.MatchedContentID,
	)
	if err != nil {
		return fmt.Errorf("failed to get default medias: %w", err)
	}

	if len(storedMedias) > 0 {
		return HandleDefaultStoredFormatDownload(
			bot, ctx, dlCtx, storedMedias,
		)
	}

	response, err := dlCtx.Extractor.Run(dlCtx)
	if err != nil {
		return fmt.Errorf("extractor fetch run failed: %w", err)
	}

	mediaList := response.MediaList
	if len(mediaList) == 0 {
		return nil
	}

	for i := range mediaList {
		defaultFormat := mediaList[i].GetDefaultFormat()
		if defaultFormat == nil {
			return fmt.Errorf("no default format found for media at index %d", i)
		}
		if len(defaultFormat.URL) == 0 {
			return fmt.Errorf("media format at index %d has no URL", i)
		}
		// ensure we can merge video and audio formats
		ensureMergeFormats(mediaList[i], defaultFormat)
		mediaList[i].Format = defaultFormat
	}

	medias, err := DownloadMedias(taskCtx, mediaList, nil)
	if err != nil {
		return fmt.Errorf("failed to download media list: %w", err)
	}

	if len(medias) == 0 {
		return errors.New("no formats downloaded")
	}

	isCaptionEnabled := true
	if dlCtx.GroupSettings != nil && !*dlCtx.GroupSettings.Captions {
		isCaptionEnabled = false
	}
	messageCaption := FormatCaption(
		mediaList[0],
		isCaptionEnabled,
	)

	// plugins act as post-processing for the media.
	// they are run after the media is downloaded
	// and before it is sent to the user
	// this allows for things like merging audio and video, etc.
	for _, media := range medias {
		for _, plugin := range media.Media.Format.Plugins {
			err = plugin(media)
			if err != nil {
				return fmt.Errorf("failed to run plugin: %w", err)
			}
		}
	}

	_, err = SendMedias(
		bot, ctx, dlCtx,
		medias,
		&models.SendMediaFormatsOptions{
			Caption:  messageCaption,
			IsStored: false,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send formats: %w", err)
	}

	return nil
}

func HandleDefaultStoredFormatDownload(
	bot *gotgbot.Bot,
	ctx *ext.Context,
	dlCtx *models.DownloadContext,
	storedMedias []*models.Media,
) error {
	isCaptionEnabled := true
	if dlCtx.GroupSettings != nil && !*dlCtx.GroupSettings.Captions {
		isCaptionEnabled = false
	}
	messageCaption := FormatCaption(
		storedMedias[0],
		isCaptionEnabled,
	)
	medias := make([]*models.DownloadedMedia, 0, len(storedMedias))
	for _, media := range storedMedias {
		medias = append(medias, &models.DownloadedMedia{
			FilePath:          "",
			ThumbnailFilePath: "",
			Media:             media,
		})
	}
	_, err := SendMedias(
		bot, ctx, dlCtx,
		medias,
		&models.SendMediaFormatsOptions{
			Caption:  messageCaption,
			IsStored: true,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to send media: %w", err)
	}
	return nil
}
