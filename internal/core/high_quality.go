package core

import (
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
)

func shouldUseHighQualityVideoUpload(chat *database.GetOrCreateChatRow, extractorID string) bool {
	if chat == nil || !chat.HighQualityVideos {
		return false
	}
	switch extractorID {
	case "instagram", "tiktok":
		return true
	default:
		return false
	}
}

func shouldUploadAsHighQualityVideo(
	chat *database.GetOrCreateChatRow,
	media *models.Media,
	format *models.MediaFormat,
) bool {
	if media == nil || format == nil {
		return false
	}
	if format.Type != database.MediaTypeVideo {
		return false
	}
	return shouldUseHighQualityVideoUpload(chat, media.ExtractorID)
}
