package models

import (
	"time"

	"gorm.io/gorm"
)

type URL struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	OriginalURL string     `json:"original_url" gorm:"not null"`
	ShortCode   string     `json:"short_code" gorm:"uniqueIndex;not null"`
	ClickCount  int        `json:"click_count" gorm:"default:0"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

type ShortenRequest struct {
	URL       string `json:"url" binding:"required"`
	ExpiresIn int    `json:"expires_in"` // in days, optional
}

type ShortenResponse struct {
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

type StatsResponse struct {
	OriginalURL string     `json:"original_url"`
	ShortCode   string     `json:"short_code"`
	ClickCount  int        `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}
