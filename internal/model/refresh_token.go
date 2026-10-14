package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken rotates on use; presenting a replaced one revokes the whole
// family (RFC 9700 4.14.2).
type RefreshToken struct {
	Base

	TokenHash     string       `gorm:"size:64;uniqueIndex;not null"`
	FamilyID      uuid.UUID    `gorm:"type:uuid;not null;index"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`
	UserID        uuid.UUID    `gorm:"type:uuid;not null;index"`
	User          *User        `gorm:"constraint:OnDelete:CASCADE"`
	CodeID        *uuid.UUID   `gorm:"type:uuid;index"`

	Scope           string    `gorm:"type:text;not null"`
	Audience        string    `gorm:"size:255;not null;default:''"`
	AuthenticatedAt time.Time `gorm:"not null"`

	// ExpiresAt is the family's: a rotated token keeps its predecessor's, so
	// rotating cannot keep a session alive for ever.
	ExpiresAt time.Time `gorm:"not null;index"`
	RevokedAt *time.Time
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

func (t RefreshToken) Usable(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
