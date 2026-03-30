package service

import (
	"fmt"
	"subtrackr/internal/models"
	"subtrackr/internal/repository"
	"time"
)

type SubscriptionService struct {
	repo            *repository.SubscriptionRepository
	categoryService *CategoryService
	userID          uint
}

func NewSubscriptionService(repo *repository.SubscriptionRepository, categoryService *CategoryService) *SubscriptionService {
	return &SubscriptionService{repo: repo, categoryService: categoryService}
}

func (s *SubscriptionService) ForUser(userID uint) *SubscriptionService {
	return &SubscriptionService{
		repo:            s.repo,
		categoryService: s.categoryService.ForUser(userID),
		userID:          userID,
	}
}

func (s *SubscriptionService) UserID() uint {
	return s.userID
}

func (s *SubscriptionService) Create(subscription *models.Subscription) (*models.Subscription, error) {
	subscription.UserID = s.userID
	if subscription.CategoryID > 0 {
		if _, err := s.categoryService.GetByID(subscription.CategoryID); err != nil {
			return nil, fmt.Errorf("invalid category")
		}
	}
	return s.repo.Create(subscription)
}

func (s *SubscriptionService) GetAll() ([]models.Subscription, error) {
	return s.repo.GetAll(s.userID)
}

func (s *SubscriptionService) GetAllSorted(sortBy, order string) ([]models.Subscription, error) {
	return s.repo.GetAllSorted(s.userID, sortBy, order)
}

func (s *SubscriptionService) GetByID(id uint) (*models.Subscription, error) {
	return s.repo.GetByID(s.userID, id)
}

func (s *SubscriptionService) Update(id uint, subscription *models.Subscription) (*models.Subscription, error) {
	subscription.UserID = s.userID
	if subscription.CategoryID > 0 {
		if _, err := s.categoryService.GetByID(subscription.CategoryID); err != nil {
			return nil, fmt.Errorf("invalid category")
		}
	}
	return s.repo.Update(s.userID, id, subscription)
}

func (s *SubscriptionService) Delete(id uint) error {
	return s.repo.Delete(s.userID, id)
}

func (s *SubscriptionService) Count() int64 {
	return s.repo.Count(s.userID)
}

func (s *SubscriptionService) GetStats() (*models.Stats, error) {
	activeSubscriptions, err := s.repo.GetActiveSubscriptions(s.userID)
	if err != nil {
		return nil, err
	}

	cancelledSubscriptions, err := s.repo.GetCancelledSubscriptions(s.userID)
	if err != nil {
		return nil, err
	}

	upcomingRenewals, err := s.repo.GetUpcomingRenewals(s.userID, 7)
	if err != nil {
		return nil, err
	}

	categoryStats, err := s.repo.GetCategoryStats(s.userID)
	if err != nil {
		return nil, err
	}

	stats := &models.Stats{
		ActiveSubscriptions:    len(activeSubscriptions),
		CancelledSubscriptions: len(cancelledSubscriptions),
		UpcomingRenewals:       len(upcomingRenewals),
		CategorySpending:       make(map[string]float64),
	}

	for _, sub := range activeSubscriptions {
		stats.TotalMonthlySpend += sub.MonthlyCost()
		stats.TotalAnnualSpend += sub.AnnualCost()
	}

	for _, sub := range cancelledSubscriptions {
		stats.TotalSaved += sub.AnnualCost()
		stats.MonthlySaved += sub.MonthlyCost()
	}

	for _, cat := range categoryStats {
		stats.CategorySpending[cat.Category] = cat.Amount
	}

	return stats, nil
}

func (s *SubscriptionService) GetAllCategories() ([]models.Category, error) {
	return s.categoryService.GetAll()
}

func (s *SubscriptionService) GetSubscriptionsNeedingReminders(reminderDays int) (map[*models.Subscription]int, error) {
	if reminderDays <= 0 {
		return make(map[*models.Subscription]int), nil
	}

	subscriptions, err := s.repo.GetUpcomingRenewals(s.userID, reminderDays)
	if err != nil {
		return nil, err
	}

	result := make(map[*models.Subscription]int)

	for i := range subscriptions {
		sub := &subscriptions[i]
		if sub.RenewalDate == nil || !sub.ReminderEnabled {
			continue
		}

		daysUntil := daysUntilDate(*sub.RenewalDate)
		if daysUntil >= 0 && daysUntil <= reminderDays {
			if sub.LastReminderRenewalDate != nil &&
				sub.RenewalDate != nil &&
				sub.LastReminderRenewalDate.Equal(*sub.RenewalDate) {
				continue
			}

			result[sub] = daysUntil
		}
	}

	return result, nil
}

func (s *SubscriptionService) GetSubscriptionsNeedingCancellationReminders(reminderDays int) (map[*models.Subscription]int, error) {
	if reminderDays <= 0 {
		return make(map[*models.Subscription]int), nil
	}

	subscriptions, err := s.repo.GetUpcomingCancellations(s.userID, reminderDays)
	if err != nil {
		return nil, err
	}

	result := make(map[*models.Subscription]int)

	for i := range subscriptions {
		sub := &subscriptions[i]
		if sub.CancellationDate == nil || !sub.ReminderEnabled {
			continue
		}

		daysUntil := daysUntilDate(*sub.CancellationDate)
		if daysUntil >= 0 && daysUntil <= reminderDays {
			if sub.LastCancellationReminderDate != nil &&
				sub.CancellationDate != nil &&
				sub.LastCancellationReminderDate.Equal(*sub.CancellationDate) {
				continue
			}

			result[sub] = daysUntil
		}
	}

	return result, nil
}

func daysUntilDate(target time.Time) int {
	now := time.Now().In(time.Local)
	target = target.In(time.Local)

	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	targetStart := time.Date(target.Year(), target.Month(), target.Day(), 0, 0, 0, 0, time.Local)

	return int(targetStart.Sub(todayStart).Hours() / 24)
}
