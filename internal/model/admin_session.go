package model

import (
	"time"

	"github.com/google/uuid"
)

type AdminSession struct {
	Base

	AdminID uuid.UUID `gorm:"type:uuid;index;not null" json:"admin_id"`
	Admin   Admin     `gorm:"foreignKey:AdminID" json:"-"`

	TokenHash string     `gorm:"uniqueIndex;size:64;not null" json:"-"`
	ExpiresAt time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`

	// IsMFAPassed records whether the second factor was cleared for this
	// session, so a half-finished login cannot be used.
	IsMFAPassed bool `gorm:"not null;default:false" json:"is_mfa_passed"`

	UserAgent string `gorm:"size:255" json:"user_agent,omitempty"`
	IP        string `gorm:"size:45" json:"ip,omitempty"`
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

func (s AdminSession) IsActive(now time.Time) bool {
	return s.RevokedAt == nil && s.IsMFAPassed && now.Before(s.ExpiresAt)
}
