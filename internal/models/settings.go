package models

import (
	"time"
)

// Settings represents application settings
type Settings struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index:idx_settings_user_key,unique,priority:1;not null;default:0"`
	Key       string    `json:"key" gorm:"index:idx_settings_user_key,unique,priority:2;not null"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// SMTPConfig represents SMTP configuration
type SMTPConfig struct {
	Host     string `json:"smtp_host"`
	Port     int    `json:"smtp_port"`
	Username string `json:"smtp_username"`
	Password string `json:"smtp_password"`
	From     string `json:"smtp_from"`
	FromName string `json:"smtp_from_name"`
	To       string `json:"smtp_to"` // Recipient email address for notifications
}

// PushoverConfig represents Pushover notification configuration
type PushoverConfig struct {
	UserKey  string `json:"pushover_user_key"`  // Pushover user key
	AppToken string `json:"pushover_app_token"` // Pushover application token
}

// TelegramConfig represents Telegram bot notification configuration
type TelegramConfig struct {
	BotToken string `json:"telegram_bot_token"`
	ChatID   string `json:"telegram_chat_id"`
}

// UIPersonalizationConfig represents frontend personalization preferences
type UIPersonalizationConfig struct {
	CustomBackgroundURL string `json:"custom_background_url"`
	EnableChibiStickers bool   `json:"enable_chibi_stickers"`
	ReduceMotion        bool   `json:"reduce_motion"`
	StaticStickersOnly  bool   `json:"static_stickers_only"`
}

// WebhookConfig represents generic webhook notification configuration
type WebhookConfig struct {
	URL     string            `json:"webhook_url"`
	Headers map[string]string `json:"webhook_headers"`
}

// NotificationSettings represents notification preferences
type NotificationSettings struct {
	RenewalReminders         bool    `json:"renewal_reminders"`
	HighCostAlerts           bool    `json:"high_cost_alerts"`
	HighCostThreshold        float64 `json:"high_cost_threshold"`
	ReminderDays             int     `json:"reminder_days"`
	CancellationReminders    bool    `json:"cancellation_reminders"`
	CancellationReminderDays int     `json:"cancellation_reminder_days"`
}

// APIKey represents an API key for external access
type APIKey struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	UserID     uint       `json:"user_id" gorm:"index;not null;default:0"`
	Name       string     `json:"name" gorm:"not null"`
	Key        string     `json:"key" gorm:"uniqueIndex;not null"`
	LastUsed   *time.Time `json:"last_used"`
	UsageCount int        `json:"usage_count" gorm:"default:0"`
	CreatedAt  time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	IsNew      bool       `json:"is_new" gorm:"-"` // Not stored in DB, just for display
}
