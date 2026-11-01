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
	// Methods are how the sign-in proved who it was (AMRPassword, …), for the
	// amr claim of every token issued in the session.
	Methods   StringList `gorm:"type:jsonb;serializer:json"`
	ExpiresAt time.Time  `gorm:"not null;index"`
	// Remember is whether the cookie was kept past the browser's closing
	// ("stay signed in"), so a step-up writes its new cookie the same way.
	Remember  bool `gorm:"not null"`
	RevokedAt *time.Time
	IP        string `gorm:"size:45;not null;default:''"`
	UserAgent string `gorm:"size:255;not null;default:''"`
}

func (UserSession) TableName() string {
	return "user_sessions"
}

func (s UserSession) Active(now time.Time) bool {
	return s.RevokedAt == nil && now.Before(s.ExpiresAt)
}
