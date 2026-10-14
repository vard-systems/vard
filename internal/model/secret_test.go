package model

import (
	"strings"
	"testing"
	"time"
)

// The pair from RFC 7636 appendix B.
func TestVerifyPKCEWithTheRFCExample(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"

	if !VerifyPKCE(challenge, verifier) {
		t.Fatal("the RFC's verifier does not match its challenge")
	}
	if VerifyPKCE(challenge, verifier[:42]+"x") {
		t.Error("a different verifier matched")
	}
	if VerifyPKCE(challenge, "short") {
		t.Error("a verifier under 43 characters matched")
	}
	if VerifyPKCE(challenge, strings.Repeat("a", 129)) {
		t.Error("a verifier over 128 characters was accepted")
	}
	if VerifyPKCE(challenge, verifier[:42]+"+") {
		t.Error("a verifier with a reserved character was accepted")
	}
}

func TestNewSecretIsHashedForStorage(t *testing.T) {
	secret, hash, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}

	if len(secret) != 43 {
		t.Errorf("secret is %d characters, want 43 (256 bits, base64url)", len(secret))
	}
	if hash != HashSecret(secret) || len(hash) != 64 || strings.Contains(hash, secret) {
		t.Errorf("hash = %q", hash)
	}

	other, _, _ := NewSecret()
	if other == secret {
		t.Error("two secrets were the same")
	}
}

func TestUsableFollowsExpiryAndUse(t *testing.T) {
	now := time.Now()
	later := now.Add(time.Minute)

	if !(AuthorizationRequest{ExpiresAt: later}).Usable(now) {
		t.Error("a fresh request is not usable")
	}
	if (AuthorizationRequest{ExpiresAt: later, CompletedAt: &now}).Usable(now) {
		t.Error("a completed request is usable")
	}
	if (RefreshToken{ExpiresAt: now}).Usable(now) {
		t.Error("a refresh token is usable at its expiry")
	}
	if (PasswordReset{ExpiresAt: later, UsedAt: &now}).Usable(now) {
		t.Error("a used reset link is usable")
	}
	if (UserSession{ExpiresAt: later, RevokedAt: &now}).Active(now) {
		t.Error("a revoked session is active")
	}
}
