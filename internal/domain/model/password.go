package model

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

// Passwords are hashed with Argon2id, in the PHC string format, with the
// parameters OWASP recommends. A bcrypt hash from before still verifies, and
// is replaced by an Argon2id one at the next sign-in that proves it.

// The lengths a password may be, in characters: a user's, an
// administrator's (whose account opens the panel), and the longest either.
const (
	MinUserPasswordLength  = 12
	MinAdminPasswordLength = 14
	MaxPasswordLength      = 128
)

// ErrPasswordTooLong is a password past MaxPasswordLength: hashing one is
// work an attacker would choose the size of.
var ErrPasswordTooLong = fmt.Errorf("password must be at most %d characters", MaxPasswordLength)

// argonParams are Argon2id's: OWASP's recommended 19 MiB, two passes, one
// lane. A stored hash names its own, so these can be raised later and the
// stored ones upgraded as people sign in.
type argonParams struct {
	memory  uint32
	time    uint32
	threads uint8
}

var currentArgon = argonParams{memory: 19 * 1024, time: 2, threads: 1}

const (
	argonSaltSize = 16
	argonKeySize  = 32
)

// hashing bounds how many hashes are computed at once, so a burst of
// sign-ins waits its turn rather than each taking its memory at the same
// moment.
var hashing = make(chan struct{}, max(2, runtime.GOMAXPROCS(0)))

func argonKey(password, salt []byte, p argonParams) []byte {
	hashing <- struct{}{}
	defer func() { <-hashing }()

	return argon2.IDKey(password, salt, p.time, p.memory, p.threads, argonKeySize)
}

// HashPassword hashes a password with Argon2id.
func HashPassword(password string) (string, error) {
	if utf8.RuneCountInString(password) > MaxPasswordLength {
		return "", ErrPasswordTooLong
	}

	salt := make([]byte, argonSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("model: password salt: %w", err)
	}

	key := argonKey([]byte(password), salt, currentArgon)
	b64 := base64.RawStdEncoding

	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version,
		currentArgon.memory, currentArgon.time, currentArgon.threads,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword says whether password is the one hash was made from, and
// whether the hash should be made again: bcrypt, or Argon2id with older
// parameters.
func VerifyPassword(hash, password string) (ok, rehash bool) {
	if utf8.RuneCountInString(password) > MaxPasswordLength {
		return false, false
	}

	switch {
	case strings.HasPrefix(hash, "$argon2id$"):
		params, salt, key, err := parseArgon(hash)
		if err != nil {
			return false, false
		}
		got := argonKey([]byte(password), salt, params)
		return subtle.ConstantTimeCompare(got, key) == 1, params != currentArgon
	case strings.HasPrefix(hash, "$2"):
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil, true
	default:
		return false, false
	}
}

var errNotArgon = errors.New("model: not an Argon2id hash")

func parseArgon(hash string) (argonParams, []byte, []byte, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argonParams{}, nil, nil, errNotArgon
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return argonParams{}, nil, nil, errNotArgon
	}

	var p argonParams
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.memory, &p.time, &p.threads); err != nil {
		return argonParams{}, nil, nil, errNotArgon
	}
	// A stored hash cannot ask for more than this server would make, or a
	// row written by someone else could make every sign-in very costly.
	if p.memory == 0 || p.memory > 256*1024 || p.time == 0 || p.time > 10 || p.threads == 0 {
		return argonParams{}, nil, nil, errNotArgon
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argonParams{}, nil, nil, errNotArgon
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(key) == 0 {
		return argonParams{}, nil, nil, errNotArgon
	}

	return p, salt, key, nil
}

var dummyHash = sync.OnceValue(func() string {
	hash, err := HashPassword("not a password anyone has")
	if err != nil {
		panic(fmt.Sprintf("model: make the dummy hash: %v", err))
	}
	return hash
})

// CompareDummyPassword does the work of checking a password against nothing,
// so refusing an unknown or throttled account takes as long as a wrong
// password and timing cannot tell which accounts exist.
func CompareDummyPassword(password string) {
	_, _ = VerifyPassword(dummyHash(), password)
}
