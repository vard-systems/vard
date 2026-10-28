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

	// PasskeyCeremony is the passkey sign-in offered to a session waiting for
	// its second factor, so only an answer to it is taken; "" for none.
	PasskeyCeremony string `gorm:"type:text;not null;default:''" json:"-"`

	// LastSeenAt is when the session was last used, to the minute: it ends
	// once AdminSecurity's idle timeout passes without use.
	LastSeenAt time.Time `gorm:"not null" json:"last_seen_at"`

	UserAgent string `gorm:"size:255;not null;default:''" json:"user_agent,omitempty"`
	IP        string `gorm:"size:45;not null;default:''" json:"ip,omitempty"`
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}

func (s AdminSession) IsActive(now time.Time) bool {
	return s.RevokedAt == nil && s.IsMFAPassed && now.Before(s.ExpiresAt)
}

// Idle says the session has gone unused for longer than timeout.
func (s AdminSession) Idle(now time.Time, timeout time.Duration) bool {
	return !s.LastSeenAt.IsZero() && now.Sub(s.LastSeenAt) > timeout
}

// SeenEvery is how often LastSeenAt is written: once a minute at most, not
// on every request.
const SeenEvery = time.Minute
