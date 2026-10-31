package model

import (
	"time"

	"github.com/google/uuid"
)

// SessionApplication says an application was given an ID token in a session,
// so that ending the session can tell it (OpenID Connect Back-Channel Logout
// 1.0). It goes when either does.
type SessionApplication struct {
	Base

	SessionID     uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_session_applications_pair"`
	Session       *UserSession `gorm:"constraint:OnDelete:CASCADE"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_session_applications_pair;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`
}

func (SessionApplication) TableName() string {
	return "session_applications"
}

// FrontchannelLogoutLifetime is how long the page that tells the browser's
// applications of a logout may be fetched.
const FrontchannelLogoutLifetime = 2 * time.Minute

// FrontchannelLogout is a browser's logout waiting for its page: the
// applications' frames to load, and where to go after. Shown once.
type FrontchannelLogout struct {
	Base

	HandleHash string   `gorm:"size:64;uniqueIndex;not null"`
	Frames     []string `gorm:"type:jsonb;serializer:json;not null"`
	Next       string   `gorm:"size:2048;not null"`
	Language   string   `gorm:"size:35;not null;default:''"`

	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (FrontchannelLogout) TableName() string {
	return "frontchannel_logouts"
}
