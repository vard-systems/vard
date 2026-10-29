package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken rotates on use; presenting a replaced one revokes the whole
// family (RFC 9700 4.14.2), unless it was replaced within RefreshReuseGrace.
type RefreshToken struct {
	Base

	TokenHash     string       `gorm:"size:64;uniqueIndex;not null"`
	FamilyID      uuid.UUID    `gorm:"type:uuid;not null;index"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`
	UserID        uuid.UUID    `gorm:"type:uuid;not null;index"`
	User          *User        `gorm:"constraint:OnDelete:CASCADE"`
	// The code, session and successor a token names may be swept before it
	// is: it outlives them, and forgets them.
	CodeID *uuid.UUID         `gorm:"type:uuid;index"`
	Code   *AuthorizationCode `gorm:"constraint:OnDelete:SET NULL"`
	// SessionID is the browser session the family began in, so an ID token
	// issued on a refresh names the same sid as the first.
	SessionID *uuid.UUID   `gorm:"type:uuid;index"`
	Session   *UserSession `gorm:"constraint:OnDelete:SET NULL"`
	// DPoPKey is the thumbprint of the key a public client's family is bound
	// to (RFC 9449 §5): a refresh needs a proof from that key. "" when unbound.
	DPoPKey string `gorm:"column:dpop_key;size:64;not null;default:''"`

	Scope           string    `gorm:"type:text;not null"`
	Audience        string    `gorm:"size:255;not null;default:''"`
	AuthenticatedAt time.Time `gorm:"not null"`

	// ExpiresAt is the family's: a rotated token keeps its predecessor's, so
	// rotating cannot keep a session alive for ever.
	ExpiresAt time.Time `gorm:"not null;index"`
	// The partial index finds the rows still holding a sealed successor, so
	// clearing them every RefreshReuseGrace reads only those.
	RevokedAt *time.Time `gorm:"index:idx_refresh_tokens_sealed,where:successor_sealed IS NOT NULL"`

	// SuccessorID is the token that replaced this one. SuccessorSealed is
	// that token itself, sealed with VARD_SECRET_KEY, so that a retry within
	// RefreshReuseGrace is answered with the same token rather than a new
	// one. It is cleared within one more grace: when read past it, at the
	// family's next rotation, or by KeepSwept every RefreshReuseGrace.
	SuccessorID     *uuid.UUID    `gorm:"type:uuid;index"`
	Successor       *RefreshToken `gorm:"foreignKey:SuccessorID;constraint:OnDelete:SET NULL"`
	SuccessorSealed []byte        `gorm:"type:bytea"`
}

func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// RefreshReuseGrace is how long a replaced refresh token may still be
// presented. A phone that loses the answer to a refresh retries with the
// token it still holds, and two tabs may refresh at once; without the grace
// either would look like theft and sign the person out. Inside it the retry
// gets the token that replaced it — the same one, so every copy the client
// keeps stays good — and a thief's copy still meets reuse detection once the
// grace is over.
const RefreshReuseGrace = 30 * time.Second

// InGrace says a token replaced by rotation, not revoked, is still within
// RefreshReuseGrace.
func (t RefreshToken) InGrace(now time.Time) bool {
	return t.RevokedAt != nil && t.SuccessorID != nil && len(t.SuccessorSealed) > 0 &&
		now.Sub(*t.RevokedAt) < RefreshReuseGrace && now.Before(t.ExpiresAt)
}

func (t RefreshToken) Usable(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
