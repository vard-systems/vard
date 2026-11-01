package model

import (
	"time"

	"github.com/google/uuid"
)

// TokenRevocation ends the access tokens issued in something that has ended:
// a session signed out, a refresh family revoked, a user's grant to an
// application taken back, or one token by its jti. An access token is a JWT
// an API checks on its own; this server's userinfo, introspection and its
// own APIs also look here. A revocation is kept as long as a token it ends
// could still be unexpired.
type TokenRevocation struct {
	Base

	// Key names what was revoked: RevokedSession, RevokedFamily,
	// RevokedGrant or RevokedToken of an id.
	Key       string    `gorm:"size:160;not null;uniqueIndex"`
	RevokedAt time.Time `gorm:"not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
}

func (TokenRevocation) TableName() string {
	return "token_revocations"
}

// The keys a revocation is kept under, and an access token looked up by.
func RevokedSession(session uuid.UUID) string { return "session:" + session.String() }

func RevokedFamily(family uuid.UUID) string { return "family:" + family.String() }

func RevokedGrant(user uuid.UUID, clientID string) string {
	return "grant:" + user.String() + ":" + clientID
}

func RevokedToken(jti string) string { return "jti:" + jti }

// NewTokenRevocation is a revocation at `at`, remembered as long as any token
// issued before it can live.
func NewTokenRevocation(key string, at time.Time) TokenRevocation {
	return TokenRevocation{Key: key, RevokedAt: at, ExpiresAt: at.Add(MaxAccessTokenLifetime * time.Second)}
}
