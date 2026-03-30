package service

import (
	"errors"
	"fmt"
	"strings"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) Count() (int64, error) {
	return s.repo.Count()
}

func (s *UserService) HasUsers() bool {
	count, err := s.repo.Count()
	return err == nil && count > 0
}

func (s *UserService) Create(username, password string) (*models.User, error) {
	username = strings.TrimSpace(username)

	if username == "" || password == "" {
		return nil, errors.New("用户名和密码不能为空")
	}

	if len(password) < 8 {
		return nil, errors.New("密码至少需要 8 位")
	}

	if _, err := s.repo.GetByUsername(username); err == nil {
		return nil, errors.New("用户名已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	email := strings.ToLower(fmt.Sprintf("%s-%d@local.subtrackr", username, time.Now().UnixNano()))

	user := &models.User{
		Username:     username,
		Email:        email,
		PasswordHash: string(passwordHash),
	}

	return s.repo.Create(user)
}

func (s *UserService) Authenticate(identifier, password string) (*models.User, error) {
	identifier = strings.TrimSpace(identifier)

	var (
		user *models.User
		err  error
	)

	if strings.Contains(identifier, "@") {
		user, err = s.repo.GetByEmail(strings.ToLower(identifier))
	} else {
		user, err = s.repo.GetByUsername(identifier)
	}
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return user, nil
}

func (s *UserService) GetByID(id uint) (*models.User, error) {
	return s.repo.GetByID(id)
}

func (s *UserService) ListAll() ([]models.User, error) {
	return s.repo.ListAll()
}

func (s *UserService) AssignOrphanedRecords(userID uint) error {
	return s.repo.AssignOrphanedRecords(userID)
}
