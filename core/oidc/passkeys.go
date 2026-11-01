package oidc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vard/core/model"
	"vard/tools/jose"
)

// Passkeys, through go-webauthn, and keeping factors sealed: a passkey's
// credential and an authenticator's seed are both stored only as the Sealer
// leaves them.

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
		if err := s.unsealJSON(factor.ID, factor.Secret, &credential); err != nil {
			return nil, err
		}
		credentials = append(credentials, credential)
	}

	return credentials, nil
}

func (s *Service) checkPasskey(ctx context.Context, user *model.User, pending *model.PendingSignIn, response []byte, now time.Time) (bool, error) {
	// The ceremony is spent on its first answer, right or wrong: a retry
	// answers the fresh one the page asks for.
	offered, err := s.store.TakePasskey(ctx, pending.ID)
	if err != nil || offered == "" {
		return false, err
	}

	var ceremony webauthn.SessionData
	if err := json.Unmarshal([]byte(offered), &ceremony); err != nil {
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

	sealed, err := s.sealJSON(factors[index].ID, credential)
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

// factorSecret is where a user's factor keeps its sealed secret.
func factorSecret(factor uuid.UUID) jose.Context {
	return model.UserFactor{Base: model.Base{ID: factor}}.SecretWhere()
}

func (s *Service) seal(factor uuid.UUID, plain []byte) (string, error) {
	sealed, err := s.sealer.SealFor(factorSecret(factor), plain)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *Service) unseal(factor uuid.UUID, stored string) ([]byte, error) {
	sealed, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return nil, fmt.Errorf("oidc: a stored factor is not base64: %w", err)
	}

	plain, err := s.sealer.OpenFor(factorSecret(factor), sealed)
	return plain, err
}

func (s *Service) sealJSON(factor uuid.UUID, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return s.seal(factor, encoded)
}

func (s *Service) unsealJSON(factor uuid.UUID, stored string, into any) error {
	plain, err := s.unseal(factor, stored)
	if err != nil {
		return err
	}

	return json.Unmarshal(plain, into)
}
