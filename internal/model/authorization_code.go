package model

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AuthorizationCodeLifetime is how long a client has to exchange a code.
// RFC 6749 recommends ten minutes at most; a client exchanges it at once.
const AuthorizationCodeLifetime = 2 * time.Minute

type AuthorizationCode struct {
	Base

	CodeHash      string       `gorm:"size:64;uniqueIndex;not null"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`
	UserID        uuid.UUID    `gorm:"type:uuid;not null;index"`
	User          *User        `gorm:"constraint:OnDelete:CASCADE"`
	SessionID     *uuid.UUID   `gorm:"type:uuid;index"`
	Session       *UserSession `gorm:"constraint:OnDelete:SET NULL"`

	RedirectURI         string    `gorm:"size:512;not null"`
	Scope               string    `gorm:"type:text;not null"`
	Nonce               string    `gorm:"type:text;not null;default:''"`
	CodeChallenge       string    `gorm:"size:128;not null;default:''"`
	CodeChallengeMethod string    `gorm:"size:8;not null;default:''"`
	Audience            string    `gorm:"size:255;not null;default:''"`
	AuthenticatedAt     time.Time `gorm:"not null"`

	ExpiresAt time.Time `gorm:"not null;index"`
	// UsedAt is set on exchange. A second presentation is refused and revokes
	// what the first issued (RFC 6749 4.1.2).
	UsedAt *time.Time
}

func (AuthorizationCode) TableName() string {
	return "authorization_codes"
}

// PKCE methods. Only S256 is accepted: "plain" protects nothing against an
// attacker who can read the authorization request (RFC 9700 section 2.1.1).
const PKCES256 = "S256"

// VerifyPKCE checks a verifier (43 to 128 unreserved characters, RFC 7636 4.1)
// against the S256 challenge.
func VerifyPKCE(challenge, verifier string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}

	for _, r := range verifier {
		unreserved := r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || strings.ContainsRune("-._~", r)
		if !unreserved {
			return false
		}
	}

	sum := sha256.Sum256([]byte(verifier))

	return base64.RawURLEncoding.EncodeToString(sum[:]) == challenge
}
