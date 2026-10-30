package model

import (
	"time"

	"github.com/google/uuid"
)

const AuthorizationRequestLifetime = 30 * time.Minute

type AuthorizationRequest struct {
	Base

	HandleHash    string       `gorm:"size:64;uniqueIndex;not null"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`

	RedirectURI         string `gorm:"size:512;not null"`
	Scope               string `gorm:"type:text;not null"`
	State               string `gorm:"type:text;not null;default:''"`
	Nonce               string `gorm:"type:text;not null;default:''"`
	CodeChallenge       string `gorm:"size:128;not null;default:''"`
	CodeChallengeMethod string `gorm:"size:8;not null;default:''"`
	Audience            string `gorm:"size:255;not null;default:''"`
	LoginHint           string `gorm:"size:255;not null;default:''"`
	// ForceConsent is prompt=consent: the consent page is shown even when
	// the user has allowed everything asked before.
	ForceConsent bool `gorm:"not null"`

	ExpiresAt   time.Time `gorm:"not null;index"`
	CompletedAt *time.Time
}

func (AuthorizationRequest) TableName() string {
	return "authorization_requests"
}

func (r AuthorizationRequest) Usable(now time.Time) bool {
	return r.CompletedAt == nil && now.Before(r.ExpiresAt)
}
