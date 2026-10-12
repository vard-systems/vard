// Package totp implements RFC 6238 codes with the parameters every
// authenticator app assumes (SHA-1, six digits, thirty seconds). They are fixed
// because an app given others would show wrong codes.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	Period = 30 * time.Second
	Digits = 6
)

// Skew accepts one step either side, for clock drift and codes typed as they
// change.
const Skew = 1

var encoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func NewSecret() (string, error) {
	var b [20]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("totp: generate secret: %w", err)
	}

	return encoding.EncodeToString(b[:]), nil
}

func Step(at time.Time) int64 {
	return at.Unix() / int64(Period/time.Second)
}

func Code(secret string, step int64) (string, error) {
	key, err := encoding.DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", fmt.Errorf("totp: secret is not base32: %w", err)
	}

	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))

	mac := hmac.New(sha1.New, key)
	mac.Write(counter[:])
	sum := mac.Sum(nil)

	// Dynamic truncation, RFC 4226 section 5.3.
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	return fmt.Sprintf("%0*d", Digits, value%1_000_000), nil
}

// Verify checks a code around `at` and returns the matched step. Steps at or
// before `after` are refused, so a code cannot be replayed.
func Verify(secret, code string, at time.Time, after int64) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != Digits {
		return 0, false
	}

	now := Step(at)
	for step := now - Skew; step <= now+Skew; step++ {
		if step <= after {
			continue
		}

		want, err := Code(secret, step)
		if err != nil {
			return 0, false
		}

		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return step, true
		}
	}

	return 0, false
}

func URI(issuer, account, secret string) string {
	label := url.PathEscape(issuer) + ":" + url.PathEscape(account)

	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", fmt.Sprint(Digits))
	query.Set("period", fmt.Sprint(int(Period/time.Second)))

	return "otpauth://totp/" + label + "?" + query.Encode()
}
