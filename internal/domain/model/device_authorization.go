package model

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The device authorization grant (RFC 8628): a device with no browser, or
// none worth typing into, shows a short code; the person enters it on their
// phone or computer, signs in and allows the application there, while the
// device asks the token endpoint until they have.

const (
	DeviceAuthorizationLifetime = 10 * time.Minute
	// DeviceInterval is how many seconds a device waits between asks, to
	// start with; asking sooner adds five (RFC 8628 §3.5).
	DeviceInterval = 5
)

type DeviceStatus string

const (
	DevicePending  DeviceStatus = "pending"
	DeviceApproved DeviceStatus = "approved"
	DeviceDenied   DeviceStatus = "denied"
)

type DeviceAuthorization struct {
	Base

	DeviceCodeHash string `gorm:"size:64;uniqueIndex;not null"`
	// UserCodeHash is the user code keyed with the server's secret: it is
	// short enough to type, so a bare hash would give it up.
	UserCodeHash string `gorm:"size:64;uniqueIndex;not null"`

	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`
	Scope         string       `gorm:"type:text;not null"`
	Audience      string       `gorm:"size:255;not null;default:''"`

	Status   DeviceStatus `gorm:"type:varchar(16);not null;check:chk_device_authorizations_status,status IN ('pending','approved','denied')"`
	Interval int          `gorm:"not null"`

	// Who allowed it, from which session, set when it is approved.
	UserID          *uuid.UUID   `gorm:"type:uuid;index"`
	User            *User        `gorm:"constraint:OnDelete:CASCADE"`
	SessionID       *uuid.UUID   `gorm:"type:uuid;index"`
	Session         *UserSession `gorm:"constraint:OnDelete:SET NULL"`
	AuthenticatedAt *time.Time
	Methods         StringList `gorm:"type:jsonb;serializer:json"`

	LastPolledAt *time.Time
	ExpiresAt    time.Time `gorm:"not null;index"`
	UsedAt       *time.Time
}

func (DeviceAuthorization) TableName() string {
	return "device_authorizations"
}

// userCodeAlphabet has no vowels, so no code spells a word, and nothing that
// reads as another letter (RFC 8628 §6.1). Eight of its twenty letters are
// about 34 bits; entering codes is rate-limited and each lives ten minutes.
const userCodeAlphabet = "BCDFGHJKLMNPQRSTVWXZ"

// NewUserCode is a code to read off a screen: two groups of four.
func NewUserCode() (string, error) {
	const whole = 256 - 256%len(userCodeAlphabet)

	code := make([]byte, 0, 8)
	for len(code) < 8 {
		var b [16]byte
		if _, err := rand.Read(b[:]); err != nil {
			return "", fmt.Errorf("generate user code: %w", err)
		}
		for _, v := range b {
			if int(v) >= whole || len(code) == 8 {
				continue
			}
			code = append(code, userCodeAlphabet[int(v)%len(userCodeAlphabet)])
		}
	}

	return string(code[:4]) + "-" + string(code[4:]), nil
}

// NormalizeUserCode is a typed code as it is compared: upper case, without
// the dash or spaces.
func NormalizeUserCode(code string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToUpper(strings.TrimSpace(code)))
}
