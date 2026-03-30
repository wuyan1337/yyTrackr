package repository

import (
	"strings"
	"subtrackr/internal/models"
	"time"

	"gorm.io/gorm"
)

type SubscriptionRepository struct {
	db              *gorm.DB
	hasLegacyColumn *bool
}

func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) checkLegacyColumn() bool {
	if r.hasLegacyColumn != nil {
		return *r.hasLegacyColumn
	}

	var exists bool
	r.db.Raw("SELECT COUNT(*) > 0 FROM pragma_table_info('subscriptions') WHERE name='category'").Scan(&exists)
	r.hasLegacyColumn = &exists
	return exists
}

func (r *SubscriptionRepository) Create(subscription *models.Subscription) (*models.Subscription, error) {
	columnExists := r.checkLegacyColumn()

	if columnExists && subscription.CategoryID > 0 {
		var category models.Category
		if err := r.db.Where("user_id = ?", subscription.UserID).First(&category, subscription.CategoryID).Error; err == nil {
			err := r.db.Transaction(func(tx *gorm.DB) error {
				result := tx.Exec(`
					INSERT INTO subscriptions (
						user_id, name, cost, schedule, status, category_id, category, original_currency,
						payment_method, account, start_date, renewal_date,
						cancellation_date, url, notes, usage, date_calculation_version, reminder_enabled, created_at, updated_at
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
					subscription.UserID, subscription.Name, subscription.Cost, subscription.Schedule,
					subscription.Status, subscription.CategoryID, category.Name, subscription.OriginalCurrency,
					subscription.PaymentMethod, subscription.Account,
					subscription.StartDate, subscription.RenewalDate,
					subscription.CancellationDate, subscription.URL,
					subscription.Notes, subscription.Usage, subscription.DateCalculationVersion, subscription.ReminderEnabled,
					time.Now(), time.Now())

				if result.Error != nil {
					return result.Error
				}

				var lastID int64
				if err := tx.Raw("SELECT last_insert_rowid()").Scan(&lastID).Error; err != nil {
					return err
				}
				subscription.ID = uint(lastID)
				return nil
			})

			if err != nil {
				return nil, err
			}
			return subscription, nil
		}
	}

	if err := r.db.Create(subscription).Error; err != nil {
		return nil, err
	}
	return subscription, nil
}

func (r *SubscriptionRepository) GetAll(userID uint) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	if err := r.db.Preload("Category").Where("user_id = ?", userID).Order("created_at DESC").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetAllSorted(userID uint, sortBy, order string) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	query := r.db.Preload("Category").Where("subscriptions.user_id = ?", userID)

	validSortColumns := map[string]string{
		"name":         "subscriptions.name",
		"cost":         "subscriptions.cost",
		"status":       "subscriptions.status",
		"renewal_date": "subscriptions.renewal_date",
		"schedule":     "subscriptions.schedule",
		"category":     "categories.name",
		"created_at":   "subscriptions.created_at",
	}

	sortColumn, ok := validSortColumns[sortBy]
	if !ok {
		sortColumn = "subscriptions.created_at"
	}
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	if sortBy == "category" {
		query = query.Joins("LEFT JOIN categories ON subscriptions.category_id = categories.id AND categories.user_id = subscriptions.user_id")
	}

	if err := query.Order(sortColumn + " " + strings.ToUpper(order)).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetByID(userID, id uint) (*models.Subscription, error) {
	var subscription models.Subscription
	if err := r.db.Preload("Category").Where("user_id = ?", userID).First(&subscription, id).Error; err != nil {
		return nil, err
	}
	return &subscription, nil
}

func (r *SubscriptionRepository) Update(userID, id uint, subscription *models.Subscription) (*models.Subscription, error) {
	var existing models.Subscription
	if err := r.db.Where("user_id = ?", userID).First(&existing, id).Error; err != nil {
		return nil, err
	}

	columnExists := r.checkLegacyColumn()

	existing.UserID = userID
	existing.Name = subscription.Name
	existing.Cost = subscription.Cost
	existing.Schedule = subscription.Schedule
	existing.Status = subscription.Status
	existing.CategoryID = subscription.CategoryID
	existing.OriginalCurrency = subscription.OriginalCurrency
	existing.PaymentMethod = subscription.PaymentMethod
	existing.Account = subscription.Account
	existing.StartDate = subscription.StartDate
	existing.LastReminderSent = subscription.LastReminderSent
	existing.LastReminderRenewalDate = subscription.LastReminderRenewalDate
	existing.RenewalDate = subscription.RenewalDate
	existing.CancellationDate = subscription.CancellationDate
	existing.URL = subscription.URL
	existing.IconURL = subscription.IconURL
	existing.Notes = subscription.Notes
	existing.Usage = subscription.Usage
	existing.ReminderEnabled = subscription.ReminderEnabled
	existing.LastCancellationReminderSent = subscription.LastCancellationReminderSent
	existing.LastCancellationReminderDate = subscription.LastCancellationReminderDate

	if columnExists && subscription.CategoryID > 0 {
		var category models.Category
		if err := r.db.Where("user_id = ?", userID).First(&category, subscription.CategoryID).Error; err == nil {
			updates := map[string]interface{}{
				"user_id":                        existing.UserID,
				"name":                           existing.Name,
				"cost":                           existing.Cost,
				"schedule":                       existing.Schedule,
				"status":                         existing.Status,
				"category_id":                    existing.CategoryID,
				"category":                       category.Name,
				"original_currency":              existing.OriginalCurrency,
				"payment_method":                 existing.PaymentMethod,
				"account":                        existing.Account,
				"start_date":                     existing.StartDate,
				"renewal_date":                   existing.RenewalDate,
				"cancellation_date":              existing.CancellationDate,
				"url":                            existing.URL,
				"icon_url":                       existing.IconURL,
				"notes":                          existing.Notes,
				"usage":                          existing.Usage,
				"last_reminder_sent":             existing.LastReminderSent,
				"last_reminder_renewal_date":     existing.LastReminderRenewalDate,
				"last_cancellation_reminder_sent": existing.LastCancellationReminderSent,
				"last_cancellation_reminder_date": existing.LastCancellationReminderDate,
				"reminder_enabled":               existing.ReminderEnabled,
				"updated_at":                     time.Now(),
			}
			if err := r.db.Model(&existing).Where("user_id = ? AND id = ?", userID, id).Updates(updates).Error; err != nil {
				return nil, err
			}
			return r.GetByID(userID, id)
		}
	}

	if err := r.db.Save(&existing).Error; err != nil {
		return nil, err
	}
	return r.GetByID(userID, id)
}

func (r *SubscriptionRepository) Delete(userID, id uint) error {
	return r.db.Where("user_id = ? AND id = ?", userID, id).Delete(&models.Subscription{}, id).Error
}

func (r *SubscriptionRepository) Count(userID uint) int64 {
	var count int64
	r.db.Model(&models.Subscription{}).Where("user_id = ?", userID).Count(&count)
	return count
}

func (r *SubscriptionRepository) GetActiveSubscriptions(userID uint) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	if err := r.db.Preload("Category").Where("user_id = ? AND status = ?", userID, "Active").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetCancelledSubscriptions(userID uint) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	if err := r.db.Preload("Category").Where("user_id = ? AND status = ?", userID, "Cancelled").Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetUpcomingRenewals(userID uint, days int) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	endDate := time.Now().AddDate(0, 0, days)

	if err := r.db.Where("user_id = ? AND status = ? AND renewal_date IS NOT NULL AND renewal_date BETWEEN ? AND ?",
		userID, "Active", time.Now(), endDate).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetUpcomingCancellations(userID uint, days int) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	endDate := time.Now().AddDate(0, 0, days)

	if err := r.db.Where("user_id = ? AND status = ? AND cancellation_date IS NOT NULL AND cancellation_date BETWEEN ? AND ?",
		userID, "Cancelled", time.Now(), endDate).Find(&subscriptions).Error; err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (r *SubscriptionRepository) GetCategoryStats(userID uint) ([]models.CategoryStat, error) {
	var stats []models.CategoryStat
	if err := r.db.Table("subscriptions").
		Select("categories.name as category, SUM(CASE WHEN subscriptions.schedule = 'Annual' THEN subscriptions.cost/12 WHEN subscriptions.schedule = 'Quarterly' THEN subscriptions.cost/3 WHEN subscriptions.schedule = 'Monthly' THEN subscriptions.cost WHEN subscriptions.schedule = 'Weekly' THEN subscriptions.cost*4.33 WHEN subscriptions.schedule = 'Daily' THEN subscriptions.cost*30.44 ELSE subscriptions.cost END) as amount, COUNT(*) as count").
		Joins("left join categories on subscriptions.category_id = categories.id AND categories.user_id = subscriptions.user_id").
		Where("subscriptions.user_id = ? AND subscriptions.status = ?", userID, "Active").
		Group("categories.name").
		Scan(&stats).Error; err != nil {
		return nil, err
	}
	return stats, nil
}
