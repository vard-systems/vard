package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vard/internal/brand"
	"vard/internal/model"
)

// Passkeys, through go-webauthn, and keeping factors sealed: a passkey's
// credential and an authenticator's seed are both stored only as the Sealer
// leaves them.

// newWebAuthn is the passkey relying party: the sign-in pages' own host and
// origin, which is where every ceremony runs.
func newWebAuthn(accountURL string) (*webauthn.WebAuthn, error) {
	parsed, err := url.Parse(accountURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, fmt.Errorf("oidc: the account URL %q has no host for passkeys", accountURL)
	}

	return webauthn.New(&webauthn.Config{
		RPID:          parsed.Hostname(),
		RPDisplayName: brand.Name,
		RPOrigins:     []string{parsed.Scheme + "://" + parsed.Host},
	})
}

// passkeyUser is a user as go-webauthn asks for one.
type passkeyUser struct {
	user        *model.User
	credentials []webauthn.Credential
}

func (p passkeyUser) WebAuthnID() []byte { return p.user.ID[:] }

func (p passkeyUser) WebAuthnName() string { return p.user.Email }

func (p passkeyUser) WebAuthnDisplayName() string {
	if name := p.user.FullName(); name != "" {
		return name
	}
	return p.user.Email
}

func (p passkeyUser) WebAuthnCredentials() []webauthn.Credential { return p.credentials }

func (s *Service) passkeyCredentials(factors []model.UserFactor) ([]webauthn.Credential, error) {
	credentials := []webauthn.Credential{}
	for _, factor := range factors {
		if factor.Method != model.MFAMethodWebAuthn || !factor.IsConfirmed() {
			continue
		}

		var credential webauthn.Credential
		if err := s.unsealJSON(factor.Secret, &credential); err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}

	return credentials, nil
}

func (s *Service) checkPasskey(ctx context.Context, user *model.User, pending *model.PendingSignIn, response []byte, now time.Time) (bool, error) {
	if pending.Passkey == "" {
		return false, nil
	}

	var ceremony webauthn.SessionData
	if err := json.Unmarshal([]byte(pending.Passkey), &ceremony); err != nil {
		return false, err
	}

	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return false, nil
	}

	factors, err := s.confirmedFactors(ctx, user.ID)
	if err != nil {
		return false, err
	}
	credentials, err := s.passkeyCredentials(factors)
	if err != nil {
		return false, err
	}

	credential, err := s.webauthn.ValidateLogin(passkeyUser{user, credentials}, ceremony, parsed)
	if err != nil {
		return false, nil
	}

	// A counter that went backwards means two copies of the key: refused,
	// as the specification suggests, rather than trusted.
	if credential.Authenticator.CloneWarning {
		s.log.Warn("a passkey's counter went backwards; refusing it", "user", user.ID)
		return false, nil
	}

	id := base64.RawURLEncoding.EncodeToString(credential.ID)
	index := slices.IndexFunc(factors, func(f model.UserFactor) bool { return f.CredentialID == id })
	if index < 0 {
		return false, nil
	}

	sealed, err := s.sealJSON(credential)
	if err != nil {
		return false, err
	}

	return true, s.store.UsedPasskey(ctx, factors[index].ID, sealed, now)
}

func (s *Service) confirmedFactors(ctx context.Context, user uuid.UUID) ([]model.UserFactor, error) {
	factors, err := s.store.UserFactors(ctx, user)
	if err != nil {
		return nil, err
	}

	return slices.DeleteFunc(factors, func(f model.UserFactor) bool { return !f.IsConfirmed() }), nil
}

func isAuthenticator(factor model.UserFactor) bool {
	return factor.Method == model.MFAMethodTOTP
}

func (s *Service) seal(plain []byte) (string, error) {
	sealed, err := s.sealer.SealBytes(plain)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *Service) unseal(stored string) ([]byte, error) {
	sealed, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return nil, fmt.Errorf("oidc: a stored factor is not base64: %w", err)
	}

	return s.sealer.OpenBytes(sealed)
}

func (s *Service) sealJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return s.seal(encoded)
}

func (s *Service) unsealJSON(stored string, into any) error {
	plain, err := s.unseal(stored)
	if err != nil {
		return err
	}

	return json.Unmarshal(plain, into)
}
