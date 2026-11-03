package model

import (
	"crypto/hmac"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"vard/tools/jose"
)

// A known browser is one that has signed in to an account before. It holds a
// cookie naming the account by a keyed hash, and its attempts at that account
// are counted on a budget of its own instead of the address's, so somebody
// else's failures at the address never keep the owner waiting in a browser
// they have used (OWASP's device cookies). The cookie proves only that it was
// handed to some browser by a successful sign-in: a stolen copy buys a budget
// of its own, throttled like any other, and no way past the password.

// KnownBrowserLifetime is how long a browser stays known after it last signed
// in to the account.
const KnownBrowserLifetime = 180 * 24 * time.Hour

// knownBrowserSkew is how far in the future an issue time may be, for
// servers whose clocks differ a little.
const knownBrowserSkew = time.Minute

// NewKnownBrowser is the cookie value for a browser that has just signed in
// to `account` (a normalised address, or an administrator's username):
// "<id>.<issued unix>.<tag>". purpose keeps the users' cookies from passing
// for the administrators'.
func NewKnownBrowser(sealer *jose.Sealer, purpose, account string, now time.Time) (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", fmt.Errorf("generate a known browser: %w", err)
	}

	payload := base64.RawURLEncoding.EncodeToString(id[:]) + "." + strconv.FormatInt(now.Unix(), 10)
	return payload + "." + base64.RawURLEncoding.EncodeToString(knownBrowserTag(sealer, purpose, payload, account)), nil
}

// KnownBrowser is the id of the browser a cookie value names, when it was
// issued for `account` within KnownBrowserLifetime; "" for anything else.
func KnownBrowser(sealer *jose.Sealer, purpose, value, account string, now time.Time) string {
	parts := strings.Split(value, ".")
	if len(parts) != 3 || parts[0] == "" {
		return ""
	}

	issued, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return ""
	}
	at := time.Unix(issued, 0)
	if at.After(now.Add(knownBrowserSkew)) || now.Sub(at) > KnownBrowserLifetime {
		return ""
	}

	tag, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(tag, knownBrowserTag(sealer, purpose, parts[0]+"."+parts[1], account)) {
		return ""
	}

	return parts[0]
}

func knownBrowserTag(sealer *jose.Sealer, purpose, payload, account string) []byte {
	return sealer.Tag(purpose, []byte(payload+"\x00"+account))
}
