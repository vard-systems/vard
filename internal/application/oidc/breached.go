package oidc

import (
	"bufio"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

// A new password may be checked against the passwords known from breaches,
// with the Pwned Passwords range API: only the first five hex characters of
// its SHA-1 leave the server (k-anonymity), and the answer lists every
// suffix sharing them. It is off unless VARD_PWNED_PASSWORDS_URL names the
// API, and an API that does not answer in time lets the password through:
// the check is advice, never a way to stop people setting one.

const breachTimeout = 2 * time.Second

// ErrPasswordBreached is a new password found among breached ones.
var ErrPasswordBreached = problem("password_breached")

// PasswordBreached says whether a password is known from a breach; false
// when the check is off or the API does not answer.
func (s *Service) PasswordBreached(ctx context.Context, password string) bool {
	if s.pwnedURL == "" {
		return false
	}

	sum := sha1.Sum([]byte(password))
	digest := strings.ToUpper(hex.EncodeToString(sum[:]))
	prefix, suffix := digest[:5], digest[5:]

	ask, cancel := context.WithTimeout(ctx, breachTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ask, http.MethodGet, strings.TrimRight(s.pwnedURL, "/")+"/range/"+prefix, nil)
	if err != nil {
		return false
	}
	// The API pads its answers so their size says nothing about the prefix.
	request.Header.Set("Add-Padding", "true")

	response, err := s.breachClient.Do(request)
	if err != nil {
		s.log.Warn("the breached password check did not answer; letting the password through", "error", err)
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}

	lines := bufio.NewScanner(response.Body)
	for lines.Scan() {
		found, count, ok := strings.Cut(strings.TrimSpace(lines.Text()), ":")
		if ok && strings.EqualFold(found, suffix) && count != "0" {
			return true
		}
	}

	return false
}

// refuseBreached is ErrPasswordBreached when a password a person is choosing
// is known from a breach.
func (s *Service) refuseBreached(ctx context.Context, password string) error {
	if s.PasswordBreached(ctx, password) {
		return ErrPasswordBreached
	}
	return nil
}
