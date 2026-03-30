package models

import "time"

// Category represents a subscription category
type Category struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id" gorm:"index:idx_categories_user_name,unique,priority:1;not null;default:0"`
	Name      string    `json:"name" gorm:"index:idx_categories_user_name,unique,priority:2;not null"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
