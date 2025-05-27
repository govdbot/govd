package database

import "github.com/govdbot/govd/models"

func GetUser(
	userID int64,
) (*models.User, error) {
	var user models.User
	err := DB.
		Where(&models.User{
			UserID: userID,
		}).
		FirstOrCreate(&user).
		Error
	if err != nil {
		return nil, err
	}
	go UpdateUserStatus(userID)
	return &user, nil
}

func UpdateUserStatus(
	userID int64,
) error {
	err := DB.
		Model(&models.User{}).
		Where(&models.User{
			UserID: userID,
		}).
		Updates(&models.User{
			LastUsed: DB.NowFunc(),
		}).
		Error
	if err != nil {
		return err
	}
	return nil
}
