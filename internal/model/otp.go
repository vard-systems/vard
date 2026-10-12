package model

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OTPSettings struct {
	Base

	CodeLength int `gorm:"not null" json:"code_length"`

	LifetimeMinutes int `gorm:"not null" json:"lifetime_minutes"`

	MaxAttempts int `gorm:"not null" json:"max_attempts"`

	ResendSeconds int `gorm:"not null" json:"resend_seconds"`
}

func (OTPSettings) TableName() string {
	return "otp_settings"
}

const (
	MinOTPLength = 4
	MaxOTPLength = 10

	MaxOTPLifetimeMinutes = 60

	MaxOTPAttempts      = 10
	MaxOTPResendSeconds = 900
)

func DefaultOTPSettings() OTPSettings {
	return OTPSettings{
		CodeLength:      6,
		LifetimeMinutes: 10,
		MaxAttempts:     5,
		ResendSeconds:   60,
	}
}

func (o OTPSettings) Lifetime() time.Duration {
	return time.Duration(o.LifetimeMinutes) * time.Minute
}

func (o OTPSettings) Resend() time.Duration {
	return time.Duration(o.ResendSeconds) * time.Second
}

func (o OTPSettings) Validate() error {
	switch {
	case o.CodeLength < MinOTPLength || o.CodeLength > MaxOTPLength:
		return fmt.Errorf("code_length must be between %d and %d", MinOTPLength, MaxOTPLength)
	case o.LifetimeMinutes < 1 || o.LifetimeMinutes > MaxOTPLifetimeMinutes:
		return fmt.Errorf("lifetime_minutes must be between 1 and %d", MaxOTPLifetimeMinutes)
	case o.MaxAttempts < 1 || o.MaxAttempts > MaxOTPAttempts:
		return fmt.Errorf("max_attempts must be between 1 and %d", MaxOTPAttempts)
	case o.ResendSeconds < 0 || o.ResendSeconds > MaxOTPResendSeconds:
		return fmt.Errorf("resend_seconds must be between 0 and %d", MaxOTPResendSeconds)
	}

	return nil
}

// LoginCode is a sign-in waiting for an emailed code. The page holds only the
// handle, so a code alone is useless without the browser that asked for it.
type LoginCode struct {
	Base

	HandleHash string `gorm:"size:64;uniqueIndex;not null"`

	CodeHash string `gorm:"size:64;not null"`

	UserID uuid.UUID `gorm:"type:uuid;not null;index"`
	User   *User     `gorm:"constraint:OnDelete:CASCADE"`

	Request string `gorm:"size:64"`

	RememberMe bool `gorm:"not null"`

	// Attempts is how many codes have been typed: every one is counted before
	// it is compared, the right one included.
	Attempts int `gorm:"not null"`

	SentAt time.Time `gorm:"not null"`

	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (LoginCode) TableName() string {
	return "login_codes"
}

func (c LoginCode) Usable(now time.Time, maxAttempts int) bool {
	return c.UsedAt == nil && now.Before(c.ExpiresAt) && c.Attempts < maxAttempts
}

func NewOTP(length int) (code, hash string, err error) {
	var b strings.Builder
	for range length {
		digit, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", "", fmt.Errorf("generate a one-time code: %w", err)
		}
		b.WriteString(digit.String())
	}

	code = b.String()

	return code, HashSecret(code), nil
}

// MatchesOTP reports whether a typed code is the one that was sent, comparing
// the hashes in constant time so nothing is learnt from how long it took.
func (c LoginCode) MatchesOTP(typed string) bool {
	typed = strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, typed)

	if typed == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(c.CodeHash), []byte(HashSecret(typed))) == 1
}
