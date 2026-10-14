package model

import (
	"time"

	"github.com/google/uuid"
)

type UserSession struct {
	Base

	TokenHash       string    `gorm:"size:64;uniqueIndex;not null"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"`
	User            *User     `gorm:"constraint:OnDelete:CASCADE"`
	AuthenticatedAt time.Time `gorm:"not null"`
	ExpiresAt       time.Time `gorm:"not null;index"`
	RevokedAt       *time.Time
	IP              string `gorm:"size:45;not null;default:''"`
	UserAgent       string `gorm:"size:255;not null;default:''"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}

func (s UserSession) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}
