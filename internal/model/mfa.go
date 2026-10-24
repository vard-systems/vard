package model

import (
	"time"

	"github.com/google/uuid"
)

type MFA struct {
	Base

	AdminID uuid.UUID `gorm:"type:uuid;index;not null" json:"admin_id"`
	Admin   Admin     `gorm:"foreignKey:AdminID" json:"-"`

	Method MFAMethod `gorm:"type:varchar(32);index;not null" json:"method"`
	Label  string    `gorm:"size:100;not null;default:''" json:"label,omitempty"`

	// Secret is the TOTP seed. Encrypt it before it is written; the database
	// should never hold it in the clear.
	Secret string `gorm:"size:255;not null" json:"-"`

	RecoveryCodes []string `gorm:"serializer:json" json:"-"`

	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	// LastUsedAt is the start of the last accepted TOTP step (UsedStep), so
	// that step is never accepted twice. Recovery codes leave it alone.
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

func (MFA) TableName() string {
	return "mfa_factors"
}

// IsConfirmed reports whether enrolment was finished. An unconfirmed factor
// must not be accepted at sign-in.
func (m MFA) IsConfirmed() bool {
	return m.ConfirmedAt != nil
}

// HasMFA reports whether the administrator has a confirmed second factor. The
// store loads only confirmed factors onto an administrator, so any is one.
func (a Admin) HasMFA() bool {
	for _, factor := range a.MFA {
		if factor.IsConfirmed() {
			return true
		}
	}
	return false
}

func (m MFA) UsedStep(period time.Duration) int64 {
	if m.LastUsedAt == nil {
		return 0
	}
	return m.LastUsedAt.Unix() / int64(period/time.Second)
}
