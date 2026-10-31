package oidc

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"vard/internal/jose"
	"vard/internal/jose/josetest"
)

// A proof brings a key of the sender's choosing, and checking a signature
// under a huge modulus costs the server, so such a proof is refused before
// it is verified, as any other bad proof is.
func TestCheckDPoPRefusesAnOversizedKey(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	s := &Service{now: func() time.Time { return now }}

	private := josetest.RSAKeyOfBits(t, 8192)
	public, err := jose.Key{Algorithm: jose.RS256, Private: private}.Public()
	if err != nil {
		t.Fatal(err)
	}
	public.KeyID, public.Use, public.Algorithm = "", "", ""

	header, _ := json.Marshal(jose.Header{Algorithm: jose.RS256, Type: dpopProofType, JWK: &public})
	claims, _ := json.Marshal(map[string]any{"jti": "one", "htm": "POST", "htu": "https://id.example/oauth2/token", "iat": now.Unix()})
	b64 := base64.RawURLEncoding
	input := b64.EncodeToString(header) + "." + b64.EncodeToString(claims)
	digest := sha256.Sum256([]byte(input))
	signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	proof := input + "." + b64.EncodeToString(signature)

	_, err = s.checkDPoP(context.Background(), proof, "POST", "https://id.example/oauth2/token", "")
	var refused *Error
	if !errors.As(err, &refused) || refused.Code != ErrInvalidDPoPProof {
		t.Fatalf("checkDPoP = %v, want %s", err, ErrInvalidDPoPProof)
	}
}
