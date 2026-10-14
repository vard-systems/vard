package model

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// Every secret given to a browser or client (code, refresh token, session
// cookie, reset link) is 256 random bits stored only as a SHA-256 hash, so a
// database copy cannot be replayed.

func NewSecret() (secret, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", fmt.Errorf("generate secret: %w", err)
	}

	secret = base64.RawURLEncoding.EncodeToString(b[:])

	return secret, HashSecret(secret), nil
}

func HashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
