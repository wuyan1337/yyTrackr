package database

import (
	"fmt"
	"log"
	"subtrackr/internal/models"

	"gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
	err := db.AutoMigrate(&models.User{}, &models.ExchangeRate{})
	if err != nil {
		return err
	}

	migrations := []func(*gorm.DB) error{
		migrateAddUserOwnership,
		migrateCategoriesToDynamic,
		migrateCurrencyFields,
		migrateDateCalculationVersioning,
		migrateSubscriptionIcons,
		migrateReminderTracking,
		migrateCancellationReminderTracking,
		migrateReminderEnabled,
	}

	for _, migration := range migrations {
		if err := migration(db); err != nil {
			return err
		}
	}

	if err := db.AutoMigrate(&models.Category{}, &models.Settings{}, &models.APIKey{}, &models.Subscription{}); err != nil {
		return err
	}
	return nil
}

func columnExists(db *gorm.DB, table, column string) bool {
	var count int64
	db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?", table), column).Scan(&count)
	return count > 0
}

func assignOwnershipToFirstUser(db *gorm.DB, table string) error {
	var firstUser models.User
	if err := db.Order("id ASC").First(&firstUser).Error; err != nil {
		return nil
	}
	return db.Exec("UPDATE "+table+" SET user_id = ? WHERE user_id IS NULL OR user_id = 0", firstUser.ID).Error
}

func migrateAddUserOwnership(db *gorm.DB) error {
	log.Println("Running migration: Adding multi-user ownership fields...")

	if !columnExists(db, "subscriptions", "user_id") {
		if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN user_id INTEGER DEFAULT 0").Error; err != nil {
			log.Printf("Note: could not add subscriptions.user_id: %v", err)
		}
	}
	if !columnExists(db, "categories", "user_id") {
		if err := db.Exec("ALTER TABLE categories ADD COLUMN user_id INTEGER DEFAULT 0").Error; err != nil {
			log.Printf("Note: could not add categories.user_id: %v", err)
		}
	}
	if !columnExists(db, "settings", "user_id") {
		if err := db.Exec("ALTER TABLE settings ADD COLUMN user_id INTEGER DEFAULT 0").Error; err != nil {
			log.Printf("Note: could not add settings.user_id: %v", err)
		}
	}
	if !columnExists(db, "api_keys", "user_id") {
		if err := db.Exec("ALTER TABLE api_keys ADD COLUMN user_id INTEGER DEFAULT 0").Error; err != nil {
			log.Printf("Note: could not add api_keys.user_id: %v", err)
		}
	}

	_ = assignOwnershipToFirstUser(db, "subscriptions")
	_ = assignOwnershipToFirstUser(db, "categories")
	_ = assignOwnershipToFirstUser(db, "settings")
	_ = assignOwnershipToFirstUser(db, "api_keys")

	db.Exec("CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_user_name ON categories(user_id, name)")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_settings_user_key ON settings(user_id, key)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id)")

	log.Println("Migration completed: multi-user ownership ready")
	return nil
}

func migrateCategoriesToDynamic(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='category'").Scan(&count)

	if count == 0 {
		return nil
	}

	log.Println("Running migration: Converting categories to dynamic system...")

	defaultCategories := []string{"Entertainment", "Productivity", "Storage", "Software", "Fitness", "Education", "Food", "Travel", "Business", "Other"}
	var categories []models.Category
	db.Find(&categories)

	if len(categories) == 0 {
		for _, name := range defaultCategories {
			db.Create(&models.Category{UserID: 0, Name: name})
		}
		db.Find(&categories)
	}

	categoryMap := make(map[string]uint)
	for _, cat := range categories {
		if cat.UserID == 0 {
			categoryMap[cat.Name] = cat.ID
		}
	}

	type OldSubscription struct {
		ID       uint
		Category string
	}

	var oldSubs []OldSubscription
	db.Table("subscriptions").Select("id, category").Scan(&oldSubs)

	for _, sub := range oldSubs {
		if sub.Category != "" {
			if catID, exists := categoryMap[sub.Category]; exists {
				db.Table("subscriptions").Where("id = ?", sub.ID).Update("category_id", catID)
			} else if otherID, exists := categoryMap["Other"]; exists {
				db.Table("subscriptions").Where("id = ?", sub.ID).Update("category_id", otherID)
			}
		}
	}

	log.Println("Migration completed: Categories converted to dynamic system")
	return nil
}

func migrateCurrencyFields(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='original_currency'").Scan(&count)
	if count > 0 {
		return nil
	}

	log.Println("Running migration: Adding currency fields...")
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN original_currency TEXT DEFAULT 'USD'").Error; err != nil {
		log.Printf("Note: Could not add original_currency column: %v", err)
	}
	if err := db.Exec("UPDATE subscriptions SET original_currency = 'USD' WHERE original_currency IS NULL OR original_currency = ''").Error; err != nil {
		log.Printf("Warning: Could not update existing subscriptions with default currency: %v", err)
	}

	log.Println("Migration completed: Currency fields added")
	return nil
}

func migrateDateCalculationVersioning(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='date_calculation_version'").Scan(&count)
	if count > 0 {
		return nil
	}

	log.Println("Running migration: Adding date calculation versioning...")
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN date_calculation_version INTEGER DEFAULT 1").Error; err != nil {
		log.Printf("Note: Could not add date_calculation_version column: %v", err)
	}
	if err := db.Exec("UPDATE subscriptions SET date_calculation_version = 1 WHERE date_calculation_version IS NULL").Error; err != nil {
		log.Printf("Warning: Could not update existing subscriptions with default version: %v", err)
	}

	log.Println("Migration completed: Date calculation versioning added")
	return nil
}

func migrateSubscriptionIcons(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='icon_url'").Scan(&count)
	if count > 0 {
		return nil
	}

	log.Println("Running migration: Adding subscription icon URLs...")
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN icon_url TEXT DEFAULT ''").Error; err != nil {
		log.Printf("Note: Could not add icon_url column: %v", err)
	}
	if err := db.Exec("UPDATE subscriptions SET icon_url = '' WHERE icon_url IS NULL").Error; err != nil {
		log.Printf("Warning: Could not update existing subscriptions with default icon_url: %v", err)
	}

	log.Println("Migration completed: Subscription icon URLs added")
	return nil
}

func migrateReminderTracking(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='last_reminder_sent'").Scan(&count)
	if count > 0 {
		return nil
	}

	log.Println("Running migration: Adding reminder tracking fields...")
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_reminder_sent DATETIME").Error; err != nil {
		log.Printf("Note: Could not add last_reminder_sent column: %v", err)
	}
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_reminder_renewal_date DATETIME").Error; err != nil {
		log.Printf("Note: Could not add last_reminder_renewal_date column: %v", err)
	}

	log.Println("Migration completed: Reminder tracking fields added")
	return nil
}

func migrateCancellationReminderTracking(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='last_cancellation_reminder_sent'").Scan(&count)
	if count > 0 {
		return nil
	}

	log.Println("Running migration: Adding cancellation reminder tracking fields...")
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_cancellation_reminder_sent DATETIME").Error; err != nil {
		log.Printf("Note: Could not add last_cancellation_reminder_sent column: %v", err)
	}
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN last_cancellation_reminder_date DATETIME").Error; err != nil {
		log.Printf("Note: Could not add last_cancellation_reminder_date column: %v", err)
	}

	log.Println("Migration completed: Cancellation reminder tracking fields added")
	return nil
}

func migrateReminderEnabled(db *gorm.DB) error {
	var count int64
	db.Raw("SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name = 'reminder_enabled'").Scan(&count)
	if count > 0 {
		return nil
	}

	log.Println("Running migration: Adding per-subscription reminder_enabled field...")
	if err := db.Exec("ALTER TABLE subscriptions ADD COLUMN reminder_enabled INTEGER DEFAULT 1").Error; err != nil {
		log.Printf("Note: Could not add reminder_enabled column: %v", err)
	}
	db.Exec("UPDATE subscriptions SET reminder_enabled = 1 WHERE reminder_enabled IS NULL")

	log.Println("Migration completed: reminder_enabled field added")
	return nil
}
