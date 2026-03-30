package service

import (
	"errors"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
)

type CategoryService struct {
	repo   *repository.CategoryRepository
	userID uint
}

func NewCategoryService(repo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) ForUser(userID uint) *CategoryService {
	return &CategoryService{
		repo:   s.repo,
		userID: userID,
	}
}

func (s *CategoryService) Create(category *models.Category) (*models.Category, error) {
	category.UserID = s.userID
	return s.repo.Create(category)
}

func (s *CategoryService) GetAll() ([]models.Category, error) {
	return s.repo.GetAll(s.userID)
}

func (s *CategoryService) GetByID(id uint) (*models.Category, error) {
	return s.repo.GetByID(s.userID, id)
}

func (s *CategoryService) Update(id uint, category *models.Category) (*models.Category, error) {
	category.UserID = s.userID
	return s.repo.Update(s.userID, id, category)
}

func (s *CategoryService) Delete(id uint) error {
	hasSubscriptions, err := s.repo.HasSubscriptions(s.userID, id)
	if err != nil {
		return err
	}
	if hasSubscriptions {
		return errors.New("cannot delete category with active subscriptions")
	}
	return s.repo.Delete(s.userID, id)
}
