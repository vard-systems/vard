package model

import (
	"time"

	"github.com/google/uuid"
)

// UserFactor is a second factor a user turned on for themselves: an
// authenticator app or a passkey. Two-step verification is the user's own
// choice; once they have a confirmed factor, every password and provider
// sign-in asks for one.
type UserFactor struct {
	Base

	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   *User     `gorm:"constraint:OnDelete:CASCADE"`

	Method MFAMethod `gorm:"type:varchar(32);not null;check:chk_user_factors_method,method IN ('totp','webauthn')"`
	// Label is what the user calls a passkey ("Work laptop").
	Label string `gorm:"size:100;not null;default:''"`

	// Secret is sealed: the authenticator's seed, or the passkey's
	// credential. While a passkey is being added it holds the ceremony's
	// session instead, until the browser answers.
	Secret string `gorm:"type:text;not null"`
	// CredentialID is a passkey's id, base64url, so a sign-in can find it;
	// "" for an authenticator app.
	CredentialID string `gorm:"size:255;not null;default:'';index"`

	// ConfirmedAt is set when setting it up is finished. An unconfirmed
	// factor is never asked for nor accepted.
	ConfirmedAt *time.Time
	// LastUsedAt is, for an authenticator, the start of the last accepted
	// step, so no code is accepted twice.
	LastUsedAt *time.Time
}

func (UserFactor) TableName() string {
	return "user_factors"
}

func (f UserFactor) IsConfirmed() bool {
	return f.ConfirmedAt != nil
}

// UsedStep is the last authenticator step accepted, 0 for none.
func (f UserFactor) UsedStep(period time.Duration) int64 {
	if f.LastUsedAt == nil {
		return 0
	}
	return f.LastUsedAt.Unix() / int64(period/time.Second)
}

// UserRecoveryCode is one of the codes a user was handed with their first
// factor, kept as a keyed hash. Each works once.
type UserRecoveryCode struct {
	Base

	UserID   uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_user_recovery_codes_code,priority:1"`
	User     *User     `gorm:"constraint:OnDelete:CASCADE"`
	CodeHash string    `gorm:"size:64;not null;uniqueIndex:idx_user_recovery_codes_code,priority:2"`
	UsedAt   *time.Time
}

func (UserRecoveryCode) TableName() string {
	return "user_recovery_codes"
}

// PendingSignInLifetime is how long a sign-in waits for its second factor.
const PendingSignInLifetime = 10 * time.Minute

// PendingSignIn is a sign-in whose first factor was accepted, waiting for the
// second. The handle is in a cookie of the browser that signed in; nothing
// else can finish it.
type PendingSignIn struct {
	Base

	HandleHash string `gorm:"size:64;uniqueIndex;not null"`

	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   *User     `gorm:"constraint:OnDelete:CASCADE"`

	// Request is the authorization request the sign-in is for, "" for the
	// account pages.
	Request    string `gorm:"size:64;not null;default:''"`
	RememberMe bool   `gorm:"not null"`

	// Passkey is the assertion ceremony offered with it, so a passkey can
	// answer; "" when the user has none.
	Passkey string `gorm:"type:text;not null;default:''"`

	// LinkConnectionID, LinkSubject and LinkEmail are a single sign-on
	// identity to link to the account once its second factor is answered:
	// the first link of a provider to an existing account is not made on the
	// provider's word alone. LinkConnectionID is nil for any other sign-in.
	LinkConnectionID *uuid.UUID     `gorm:"type:uuid"`
	LinkConnection   *SSOConnection `gorm:"constraint:OnDelete:CASCADE"`
	LinkSubject      string         `gorm:"size:512;not null;default:''"`
	LinkEmail        string         `gorm:"size:255;not null;default:''"`

	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (PendingSignIn) TableName() string {
	return "pending_sign_ins"
}

func (p PendingSignIn) Usable(now time.Time) bool {
	return p.UsedAt == nil && now.Before(p.ExpiresAt)
}

// PasskeyLogin is a sign-in with a passkey alone, waiting for the browser's
// answer: the ceremony it was offered, used once.
type PasskeyLogin struct {
	Base

	HandleHash string `gorm:"size:64;uniqueIndex;not null"`
	// Request is the authorization request it is for, "" for the account pages.
	Request    string `gorm:"size:64;not null;default:''"`
	RememberMe bool   `gorm:"not null"`
	Ceremony   string `gorm:"type:text;not null"`

	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (PasskeyLogin) TableName() string {
	return "passkey_logins"
}

// PasskeyLoginLifetime is how long the browser has to answer.
const PasskeyLoginLifetime = 5 * time.Minute
