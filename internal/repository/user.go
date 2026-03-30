package repository

import (
	"subtrackr/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(user *models.User) (*models.User, error) {
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Count() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *UserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) ListAll() ([]models.User, error) {
	var users []models.User
	if err := r.db.Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) AssignOrphanedRecords(userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, table := range []string{"subscriptions", "categories", "api_keys"} {
			if err := tx.Exec("UPDATE "+table+" SET user_id = ? WHERE user_id IS NULL OR user_id = 0", userID).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("UPDATE settings SET user_id = ? WHERE (user_id IS NULL OR user_id = 0) AND key <> ?", userID, "auth_session_secret").Error; err != nil {
			return err
		}
		return nil
	})
}
