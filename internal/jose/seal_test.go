package jose

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"testing"
)

const (
	secretNow    = "the-secret-in-use-now-0123456789abcdef"
	secretBefore = "the-secret-used-before-0123456789abcdef"
)

// A sealed value opens only where it was sealed for: the SMTP password moved
// into a refresh token's sealed successor — which would be handed out as a
// refresh token — does not open there, nor in another row of its own column.
func TestASealedValueOpensOnlyWhereItWasSealedFor(t *testing.T) {
	sealer, err := NewSealer(secretNow)
	if err != nil {
		t.Fatal(err)
	}

	here := Bind("mail_settings", "password", "0192")
	sealed, err := sealer.SealFor(here, []byte("smtp password"))
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		where Context
		opens bool
	}{
		{"where it was sealed for", here, true},
		{"another table's column", Bind("refresh_tokens", "successor_sealed", "0192"), false},
		{"another row of the same column", Bind("mail_settings", "password", "0193"), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plain, _, err := sealer.OpenFor(c.where, sealed)
			if c.opens != (err == nil) {
				t.Fatalf("OpenFor = %q, %v; want it to open: %v", plain, err, c.opens)
			}
			if c.opens && string(plain) != "smtp password" {
				t.Errorf("opened %q", plain)
			}
		})
	}
}

// A key rotated out still opens what it sealed, which is stale; it seals
// nothing new, and without it nothing it sealed opens.
func TestARotatedKeyStillOpensWhatItSealed(t *testing.T) {
	before, err := NewSealer(secretBefore)
	if err != nil {
		t.Fatal(err)
	}
	where := Bind("social_providers", "client_secret", "1")
	sealed, err := before.SealFor(where, []byte("client secret"))
	if err != nil {
		t.Fatal(err)
	}

	rotated, err := NewSealer(secretNow, secretBefore)
	if err != nil {
		t.Fatal(err)
	}
	plain, stale, err := rotated.OpenFor(where, sealed)
	if err != nil || string(plain) != "client secret" || !stale {
		t.Errorf("opened with the previous key = %q, stale %v, %v; want it, stale", plain, stale, err)
	}

	resealed, err := rotated.SealFor(where, plain)
	if err != nil {
		t.Fatal(err)
	}
	if _, stale, err := rotated.OpenFor(where, resealed); err != nil || stale {
		t.Errorf("sealed again = stale %v, %v; want fresh", stale, err)
	}
	if _, _, err := before.OpenFor(where, resealed); err == nil {
		t.Error("the previous key opened what the current one sealed")
	}

	if _, _, err := mustSealer(t, secretNow).OpenFor(where, sealed); err == nil {
		t.Error("the previous key's value opened once the previous key was dropped")
	}
}

// A value sealed before versions existed — a bare nonce and ciphertext under
// a SHA-256 of the secret — still opens, and is stale.
func TestALegacySealedValueOpens(t *testing.T) {
	sum := sha256.Sum256([]byte(secretNow))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		t.Fatal(err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, aead.NonceSize())
	_, _ = rand.Read(nonce)
	legacy := aead.Seal(nonce, nonce, []byte("from before"), nil)

	plain, stale, err := mustSealer(t, secretNow).OpenFor(Bind("mail_settings", "password", "anything"), legacy)
	if err != nil || string(plain) != "from before" || !stale {
		t.Errorf("a legacy value = %q, stale %v, %v; want it, stale", plain, stale, err)
	}
}

// A keyed hash stored before keys were derived, or under the previous key,
// is still among the ones a value is matched with: recovery codes keep
// working through the upgrade and a rotation.
func TestStoredTagsKeepMatching(t *testing.T) {
	sum := sha256.Sum256([]byte(secretBefore))
	legacyRoot := sha256.Sum256(append(sum[:], []byte("purpose")...))
	legacy := hmacSHA256(legacyRoot[:], []byte("code"))

	before := mustSealer(t, secretBefore).Tag("purpose", []byte("code"))
	rotated := mustSealer(t, secretNow, secretBefore)

	tags := rotated.Tags("purpose", []byte("code"))
	for name, stored := range map[string][]byte{"tagged before keys were derived": legacy, "tagged with the previous key": before} {
		found := false
		for _, tag := range tags {
			found = found || bytes.Equal(tag, stored)
		}
		if !found {
			t.Errorf("a value %s does not match", name)
		}
	}
	if !bytes.Equal(tags[0], rotated.Tag("purpose", []byte("code"))) {
		t.Error("the first tag is not the current key's")
	}
}

func mustSealer(t *testing.T, secret string, previous ...string) *Sealer {
	t.Helper()
	s, err := NewSealer(secret, previous...)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func hmacSHA256(key, data []byte) []byte {
	return keyedHashRaw(key, data)
}
