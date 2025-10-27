package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func StoreMedia(
	ctx context.Context,
	extractor *models.Extractor,
	media *models.Media,
	messages []gotgbot.Message,
	formats []*models.DownloadedFormat,
) error {
	if len(media.Items) == 0 {
		return fmt.Errorf("no item to store")
	}

	fileIDs, fileSizes := collectMessageData(messages)
	if len(fileIDs) != len(media.Items) {
		return fmt.Errorf("number of file IDs does not match number of media items")
	}
	if len(fileSizes) != len(media.Items) {
		return fmt.Errorf("number of file sizes does not match number of media items")
	}

	tx, err := database.Conn().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := database.Q().WithTx(tx)

	mediaID, err := qtx.CreateMedia(ctx, database.CreateMediaParams{
		ExtractorID: extractor.ID,
		ContentUrl:  media.ContentURL,
		ContentID:   media.ContentID,
		Caption: pgtype.Text{
			String: media.Caption,
			Valid:  media.Caption != "",
		},
		Nsfw: media.NSFW,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			// unique violation, media already exists
			logger.L.Debugf("media %s already exists in database", media.ContentID)
			return nil
		}
		return err
	}

	for i := range media.Items {
		itemID, err := qtx.CreateMediaItem(ctx, mediaID)
		if err != nil {
			return err
		}

		fileID := fileIDs[i]
		fileSize := fileSizes[i]
		format := formats[i].Format

		err = qtx.CreateMediaFormat(ctx, database.CreateMediaFormatParams{
			ItemID:   itemID,
			FormatID: format.FormatID,
			FileID:   fileID,
			Type:     format.Type,
			AudioCodec: database.NullMediaCodec{
				MediaCodec: format.AudioCodec,
				Valid:      format.AudioCodec != "",
			},
			VideoCodec: database.NullMediaCodec{
				MediaCodec: format.VideoCodec,
				Valid:      format.VideoCodec != "",
			},
			FileSize: pgtype.Int8{
				Int64: fileSize,
				Valid: fileSize != 0,
			},
			Duration: pgtype.Int4{
				Int32: format.Duration,
				Valid: format.Duration != 0,
			},
			Title: pgtype.Text{
				String: format.Title,
				Valid:  format.Title != "",
			},
			Artist: pgtype.Text{
				String: format.Artist,
				Valid:  format.Artist != "",
			},
			Width: pgtype.Int4{
				Int32: format.Width,
				Valid: format.Width != 0,
			},
			Height: pgtype.Int4{
				Int32: format.Height,
				Valid: format.Height != 0,
			},
			Bitrate: pgtype.Int8{
				Int64: format.Bitrate,
				Valid: format.Bitrate != 0,
			},
		})
		if err != nil {
			return err
		}
	}

	logger.L.Debugf("stored media %s with %d items", media.ContentID, len(media.Items))

	return tx.Commit(ctx)
}

func collectMessageData(messages []gotgbot.Message) ([]string, []int64) {
	fileIDs := make([]string, 0, len(messages))
	fileSizes := make([]int64, 0, len(messages))
	for _, msg := range messages {
		fileID := util.GetMessageFileID(&msg)
		fileIDs = append(fileIDs, fileID)

		fileSize := util.GetMessageFileSize(&msg)
		fileSizes = append(fileSizes, fileSize)
	}
	return fileIDs, fileSizes
}

func ParseStoredMedia(
	ctx context.Context,
	extractor *models.Extractor,
	mediaRow *database.GetMediaByContentIDRow,
) (*models.Media, error) {
	itemRows, err := database.Q().GetMediaItems(ctx, mediaRow.ID)
	if err != nil {
		return nil, err
	}
	if len(itemRows) == 0 {
		return nil, fmt.Errorf("no media items found")
	}

	items := make([]*models.MediaItem, 0, len(itemRows))
	for _, row := range itemRows {
		format, err := database.Q().GetMediaFormat(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		items = append(items, &models.MediaItem{
			Formats: []*models.MediaFormat{parseFormatFromDB(&format)},
		})
	}

	media := &models.Media{
		ContentID:   mediaRow.ContentID,
		ContentURL:  mediaRow.ContentUrl,
		Caption:     mediaRow.Caption.String,
		NSFW:        mediaRow.Nsfw,
		ExtractorID: extractor.ID,
		Items:       items,
	}

	return media, nil
}
