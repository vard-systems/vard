package model

import (
	"time"

	"github.com/google/uuid"
)

const EmailVerificationLifetime = 24 * time.Hour

type EmailVerification struct {
	Base

	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	User      *User     `gorm:"constraint:OnDelete:CASCADE"`
	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time

	NewEmail string `gorm:"size:255;not null;default:''"`
}

func (v EmailVerification) IsChange() bool {
	return v.NewEmail != ""
}

func (EmailVerification) TableName() string {
	return "email_verifications"
}

func (v EmailVerification) Usable(now time.Time) bool {
	return v.UsedAt == nil && now.Before(v.ExpiresAt)
}
