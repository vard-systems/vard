// Package passkey is the WebAuthn relying party both kinds of sign-in use:
// the sign-in pages' for users, the panel's for administrators. Each is its
// own site, so a passkey made on one is never offered on the other.
package passkey

import (
	"fmt"
	"net/url"

	"github.com/go-webauthn/webauthn/webauthn"

	"vard/core/brand"
)

// RelyingParty is the passkey relying party of the site at siteURL: its host
// and origin, where every ceremony runs.
func RelyingParty(siteURL string) (*webauthn.WebAuthn, error) {
	parsed, err := url.Parse(siteURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, fmt.Errorf("passkey: %q has no host for passkeys", siteURL)
	}

	return webauthn.New(&webauthn.Config{
		RPID:          parsed.Hostname(),
		RPDisplayName: brand.Name,
		RPOrigins:     []string{parsed.Scheme + "://" + parsed.Host},
	})
}
