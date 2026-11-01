package jose

import (
	"bytes"
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

// The installation's tag key makes a keyed hash the same under any
// VARD_SECRET_KEY, so a recovery code hashed before a rotation still matches
// after it; without it, a hash follows the secret.
func TestATagSurvivesARotation(t *testing.T) {
	tagKey := []byte("the installation's own tag key, 32b")

	before := mustSealer(t, secretBefore)
	before.UseTagKey(tagKey)
	after := mustSealer(t, secretNow, secretBefore)
	after.UseTagKey(tagKey)

	if !bytes.Equal(before.Tag("purpose", []byte("code")), after.Tag("purpose", []byte("code"))) {
		t.Error("a tag changed with the secret, though the tag key did not")
	}
	if bytes.Equal(mustSealer(t, secretBefore).Tag("purpose", []byte("code")), mustSealer(t, secretNow).Tag("purpose", []byte("code"))) {
		t.Error("without a tag key, two secrets made the same tag")
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
