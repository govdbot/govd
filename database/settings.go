package database

import (
	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/models"
)

func GetGroupSettings(
	chatID int64,
) (*models.GroupSettings, error) {
	var groupSettings models.GroupSettings
	err := DB.
		Where(&models.GroupSettings{
			ChatID: chatID,
		}).
		FirstOrCreate(&groupSettings, &models.GroupSettings{
			ChatID:          chatID,
			Captions:        &config.Env.DefaultCaptions,
			Silent:          &config.Env.DefaultSilent,
			NSFW:            &config.Env.DefaultNSFW,
			MediaGroupLimit: config.Env.DefaultMediaGroupLimit,
		}).
		Error
	if err != nil {
		return nil, err
	}
	return &groupSettings, nil
}

func UpdateGroupSettings(
	chatID int64,
	settings *models.GroupSettings,
) error {
	err := DB.
		Where(&models.GroupSettings{
			ChatID: chatID,
		}).
		Updates(settings).
		Error
	if err != nil {
		return err
	}
	return nil
}
