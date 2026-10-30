package model

import (
	"time"

	"github.com/google/uuid"
)

const PasswordResetLifetime = time.Hour

type PasswordReset struct {
	Base

	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	User      *User     `gorm:"constraint:OnDelete:CASCADE"`
	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (PasswordReset) TableName() string {
	return "password_resets"
}

func (r PasswordReset) Usable(now time.Time) bool {
	return r.UsedAt == nil && now.Before(r.ExpiresAt)
}
