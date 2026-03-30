package repository

import (
	"subtrackr/internal/models"
	"time"

	"gorm.io/gorm"
)

type SettingsRepository struct {
	db *gorm.DB
}

func NewSettingsRepository(db *gorm.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

func (r *SettingsRepository) SetForUser(userID uint, key, value string) error {
	var setting models.Settings
	err := r.db.Where("user_id = ? AND key = ?", userID, key).First(&setting).Error
	if err == gorm.ErrRecordNotFound {
		setting = models.Settings{
			UserID: userID,
			Key:    key,
			Value:  value,
		}
		return r.db.Create(&setting).Error
	} else if err != nil {
		return err
	}

	setting.Value = value
	return r.db.Save(&setting).Error
}

func (r *SettingsRepository) GetForUser(userID uint, key string) (string, error) {
	var setting models.Settings
	err := r.db.Where("user_id = ? AND key = ?", userID, key).First(&setting).Error
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

func (r *SettingsRepository) DeleteForUser(userID uint, key string) error {
	return r.db.Where("user_id = ? AND key = ?", userID, key).Delete(&models.Settings{}).Error
}

func (r *SettingsRepository) GetAllForUser(userID uint) ([]models.Settings, error) {
	var settings []models.Settings
	err := r.db.Where("user_id = ?", userID).Find(&settings).Error
	return settings, err
}

func (r *SettingsRepository) SetGlobal(key, value string) error {
	return r.SetForUser(0, key, value)
}

func (r *SettingsRepository) GetGlobal(key string) (string, error) {
	return r.GetForUser(0, key)
}

func (r *SettingsRepository) DeleteGlobal(key string) error {
	return r.DeleteForUser(0, key)
}

func (r *SettingsRepository) FindUserIDByKeyValue(key, value string) (uint, error) {
	var setting models.Settings
	if err := r.db.Where("key = ? AND value = ?", key, value).First(&setting).Error; err != nil {
		return 0, err
	}
	return setting.UserID, nil
}

func (r *SettingsRepository) CreateAPIKey(apiKey *models.APIKey) (*models.APIKey, error) {
	if err := r.db.Create(apiKey).Error; err != nil {
		return nil, err
	}
	return apiKey, nil
}

func (r *SettingsRepository) GetAllAPIKeysForUser(userID uint) ([]models.APIKey, error) {
	var keys []models.APIKey
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error
	return keys, err
}

func (r *SettingsRepository) GetAPIKeyByKey(key string) (*models.APIKey, error) {
	var apiKey models.APIKey
	err := r.db.Where("key = ?", key).First(&apiKey).Error
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

func (r *SettingsRepository) DeleteAPIKeyForUser(userID, id uint) error {
	return r.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.APIKey{}, id).Error
}

func (r *SettingsRepository) UpdateAPIKeyUsage(id uint) error {
	now := time.Now()
	return r.db.Model(&models.APIKey{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_used":   now,
		"usage_count": gorm.Expr("usage_count + ?", 1),
	}).Error
}
