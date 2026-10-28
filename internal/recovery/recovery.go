// Package recovery makes the one-time codes handed out with a second factor,
// for administrators and users alike: drawn evenly, read off paper as two
// halves, and hashed with the server's key.
package recovery

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"vard/internal/jose"
)

// Count is how many codes are handed out at once.
const Count = 10

const (
	alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	// letters is how many of them a code is made of, the dash aside. Ten of
	// this alphabet is a little under fifty bits.
	letters = 10
)

// New returns Count fresh codes and their hashes under purpose.
func New(sealer *jose.Sealer, purpose string) (codes, hashes []string, err error) {
	for range Count {
		code, err := newCode()
		if err != nil {
			return nil, nil, err
		}

		codes = append(codes, code)
		hashes = append(hashes, Hash(sealer, purpose, code))
	}

	return codes, hashes, nil
}

// Hash keys the hash with the server's secret: a code has under fifty bits,
// which a bare SHA-256 from a database dump would give up quickly. purpose
// keeps an administrator's codes and a user's apart.
func Hash(sealer *jose.Sealer, purpose, code string) string {
	return hex.EncodeToString(sealer.Tag(purpose, []byte(Normalize(code))))
}

// Hashes are every hash a stored code may be kept under: the current key's,
// a previous key's, and the one before keys were derived, so codes handed
// out before an upgrade or a key rotation still work.
func Hashes(sealer *jose.Sealer, purpose, code string) []string {
	tags := sealer.Tags(purpose, []byte(Normalize(code)))
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = hex.EncodeToString(tag)
	}
	return out
}

// Normalize is a code as it is compared: lower case, without the dash or
// spaces a reader may have typed.
func Normalize(code string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(code)))
}

// Is tells a recovery code from an authenticator's six digits by its shape.
func Is(code string) bool {
	return len(Normalize(code)) == letters
}

// newCode draws each letter uniformly, rejecting bytes past the last whole
// alphabet rather than taking a biased modulo.
func newCode() (string, error) {
	const whole = 256 - 256%len(alphabet)

	drawn := make([]byte, 0, letters)

	for len(drawn) < letters {
		var batch [letters]byte
		if _, err := rand.Read(batch[:]); err != nil {
			return "", fmt.Errorf("recovery: generate code: %w", err)
		}

		for _, v := range batch {
			if int(v) >= whole {
				continue
			}

			drawn = append(drawn, alphabet[int(v)%len(alphabet)])
			if len(drawn) == letters {
				break
			}
		}
	}

	half := letters / 2

	return string(drawn[:half]) + "-" + string(drawn[half:]), nil
}
