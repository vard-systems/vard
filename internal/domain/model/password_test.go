package model

import (
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordsAreHashedWithArgon2id(t *testing.T) {
	hash, err := HashPassword("a password of fourteen")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Errorf("hash = %q, want a PHC Argon2id string", hash)
	}

	cases := []struct {
		name               string
		password           string
		wantOK, wantRehash bool
	}{
		{"the password", "a password of fourteen", true, false},
		{"another", "another password", false, false},
		{"one past the longest", strings.Repeat("a", MaxPasswordLength+1), false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ok, rehash := VerifyPassword(hash, c.password)
			if ok != c.wantOK || rehash != c.wantRehash {
				t.Errorf("VerifyPassword = %v, %v; want %v, %v", ok, rehash, c.wantOK, c.wantRehash)
			}
		})
	}
}

// A bcrypt hash from before still verifies, and asks to be made again.
func TestABcryptHashVerifiesAndAsksToBeRehashed(t *testing.T) {
	old, err := bcrypt.GenerateFromPassword([]byte("a password from before"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}

	if ok, rehash := VerifyPassword(string(old), "a password from before"); !ok || !rehash {
		t.Errorf("the right password against bcrypt = %v, %v; want true, true", ok, rehash)
	}
	if ok, _ := VerifyPassword(string(old), "a wrong password"); ok {
		t.Error("a wrong password matched a bcrypt hash")
	}
}

// A stored hash cannot ask for work the server would never do: one naming a
// huge memory cost is refused rather than computed.
func TestAHashAskingForTooMuchIsRefused(t *testing.T) {
	if ok, _ := VerifyPassword("$argon2id$v=19$m=4194304,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$a2V5", "anything"); ok {
		t.Error("a hash naming 4 GiB of memory was computed")
	}
}

// Passwords are limited in characters, not bytes: a long passphrase in any
// script fits, and a password past the limit is refused, not cut short.
func TestPasswordLengthIsInCharacters(t *testing.T) {
	if _, err := HashPassword(strings.Repeat("ж", MaxPasswordLength)); err != nil {
		t.Errorf("a %d-character Cyrillic passphrase = %v, want it taken", MaxPasswordLength, err)
	}
	if _, err := HashPassword(strings.Repeat("a", MaxPasswordLength+1)); err != ErrPasswordTooLong {
		t.Errorf("one character past the longest = %v, want ErrPasswordTooLong", err)
	}
}

// Checking against nothing costs what checking a real password does, so
// timing cannot tell an unknown account from a wrong password.
func TestTheDummyCheckCostsWhatARealOneDoes(t *testing.T) {
	hash, err := HashPassword("a password of fourteen")
	if err != nil {
		t.Fatal(err)
	}
	CompareDummyPassword("warm up")

	measure := func(check func()) time.Duration {
		start := time.Now()
		for range 5 {
			check()
		}
		return time.Since(start)
	}
	real := measure(func() { VerifyPassword(hash, "a wrong password") })
	dummy := measure(func() { CompareDummyPassword("a wrong password") })

	if ratio := float64(dummy) / float64(real); ratio < 0.5 || ratio > 2 {
		t.Errorf("a dummy check took %v and a real one %v, want about the same", dummy, real)
	}
}
