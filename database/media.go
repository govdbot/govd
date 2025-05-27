package database

import (
	"fmt"

	"github.com/govdbot/govd/models"

	"gorm.io/gorm"
)

func GetDefaultMedias(
	extractorCodeName string,
	contentID string,
) ([]*models.Media, error) {
	var mediaList []*models.Media

	err := DB.
		Where(&models.Media{
			ExtractorCodeName: extractorCodeName,
			ContentID:         contentID,
		}).
		Preload("Format", "is_default = ?", true).
		Find(&mediaList).
		Error

	if err != nil {
		return nil, fmt.Errorf("failed to get stored media list: %w", err)
	}

	return mediaList, nil
}

func StoreMedia(
	extractorCodeName string,
	contentID string,
	media *models.Media,
) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where(models.Media{
			ExtractorCodeName: extractorCodeName,
			ContentID:         contentID,
		}).Create(&media).Error; err != nil {
			return fmt.Errorf("failed to get or create media: %w", err)
		}
		if media.Format != nil {
			format := media.Format
			format.MediaID = media.ID

			if err := tx.Where(models.MediaFormat{
				MediaID:  format.MediaID,
				FormatID: format.FormatID,
				Type:     format.Type,
			}).FirstOrCreate(format).Error; err != nil {
				return fmt.Errorf("failed to get or create format: %w", err)
			}
		}

		return nil
	})
}
