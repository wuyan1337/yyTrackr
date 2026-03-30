package repository

import (
	"subtrackr/internal/models"

	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

func (r *CategoryRepository) Create(category *models.Category) (*models.Category, error) {
	if err := r.db.Create(category).Error; err != nil {
		return nil, err
	}
	return category, nil
}

func (r *CategoryRepository) GetAll(userID uint) ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Where("user_id = ?", userID).Order("name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

func (r *CategoryRepository) GetByID(userID, id uint) (*models.Category, error) {
	var category models.Category
	if err := r.db.Where("user_id = ?", userID).First(&category, id).Error; err != nil {
		return nil, err
	}
	return &category, nil
}

func (r *CategoryRepository) Update(userID, id uint, category *models.Category) (*models.Category, error) {
	if err := r.db.Model(&models.Category{}).Where("user_id = ? AND id = ?", userID, id).Updates(category).Error; err != nil {
		return nil, err
	}
	return r.GetByID(userID, id)
}

func (r *CategoryRepository) Delete(userID, id uint) error {
	return r.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Category{}, id).Error
}

func (r *CategoryRepository) HasSubscriptions(userID, id uint) (bool, error) {
	var count int64
	err := r.db.Model(&models.Subscription{}).Where("user_id = ? AND category_id = ?", userID, id).Count(&count).Error
	return count > 0, err
}
