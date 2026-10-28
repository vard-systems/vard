package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vard/internal/model"
)

// An administrator's passkeys sit beside their authenticator, never in place
// of it: the authenticator's recovery codes are the way back for both, so a
// passkey is added with a code from it, and turning it off takes them all.

// maxAdminPasskeys is plenty for one person's devices.
const maxAdminPasskeys = 10

var (
	ErrPasskeyLimit   = errors.New("an administrator has as many passkeys as they may")
	ErrPasskeyRefused = errors.New("the passkey's answer was not accepted")
	ErrPasskeyUnknown = errors.New("no such passkey")
)

// Passkey is one of an administrator's passkeys, as their profile shows it.
type Passkey struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	AddedAt    time.Time  `json:"added_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// passkeyAdmin is an administrator as go-webauthn asks for one.
type passkeyAdmin struct {
	admin       *model.Admin
	credentials []webauthn.Credential
}

func (p passkeyAdmin) WebAuthnID() []byte                         { return p.admin.ID[:] }
func (p passkeyAdmin) WebAuthnName() string                       { return p.admin.Email }
func (p passkeyAdmin) WebAuthnDisplayName() string                { return p.admin.Email }
func (p passkeyAdmin) WebAuthnCredentials() []webauthn.Credential { return p.credentials }

// BeginPasskey starts adding a passkey, with a code from the administrator's
// authenticator (or a recovery code), and answers what
// navigator.credentials.create needs.
func (s *Service) BeginPasskey(ctx context.Context, admin *model.Admin, code string, req Request) (*protocol.PublicKeyCredentialCreationOptions, error) {
	current, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return nil, err
	}
	if _, err := s.check(ctx, admin, current, code, req); err != nil {
		return nil, err
	}

	factors, credentials, err := s.passkeys(ctx, admin)
	if err != nil {
		return nil, err
	}
	if len(factors) >= maxAdminPasskeys {
		return nil, ErrPasskeyLimit
	}

	exclusions := make([]protocol.CredentialDescriptor, len(credentials))
	for i, credential := range credentials {
		exclusions[i] = credential.Descriptor()
	}

	creation, ceremony, err := s.webauthn.BeginRegistration(passkeyAdmin{admin, credentials},
		webauthn.WithExclusions(exclusions),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementDiscouraged))
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sealed, err := s.sealJSON(id, ceremony)
	if err != nil {
		return nil, err
	}

	factor := model.MFA{AdminID: admin.ID, Method: model.MFAMethodWebAuthn, Label: "Passkey", Secret: sealed}
	factor.ID = id
	if err := s.store.StartAdminPasskey(ctx, &factor); err != nil {
		return nil, err
	}

	return &creation.Response, nil
}

// FinishPasskey keeps the passkey the browser made, under the name given.
func (s *Service) FinishPasskey(ctx context.Context, admin *model.Admin, label string, response []byte, req Request) error {
	pending, err := s.store.MFAFactors(ctx, admin.ID, model.MFAMethodWebAuthn)
	if err != nil {
		return err
	}
	index := slices.IndexFunc(pending, func(f model.MFA) bool { return !f.IsConfirmed() })
	if index < 0 {
		return ErrNotEnrolling
	}
	factor := pending[index]

	var ceremony webauthn.SessionData
	if err := s.openJSON(factor.ID, factor.Secret, &ceremony); err != nil {
		return err
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		return ErrPasskeyRefused
	}
	credential, err := s.webauthn.CreateCredential(passkeyAdmin{admin: admin}, ceremony, parsed)
	if err != nil {
		return ErrPasskeyRefused
	}

	if factor.Secret, err = s.sealJSON(factor.ID, credential); err != nil {
		return err
	}
	now := s.now()
	factor.CredentialID = base64.RawURLEncoding.EncodeToString(credential.ID)
	factor.Label = label
	factor.ConfirmedAt = &now

	if err := s.store.ConfirmAdminPasskey(ctx, &factor); err != nil {
		return err
	}

	s.recordWith(ctx, &admin.ID, admin.Username, "admin.passkey_added", req, map[string]any{"label": label})

	return nil
}

// RemovePasskey removes one of the administrator's passkeys, with a code from
// their authenticator.
func (s *Service) RemovePasskey(ctx context.Context, admin *model.Admin, id, code string, req Request) error {
	factor, err := uuid.Parse(id)
	if err != nil {
		return ErrPasskeyUnknown
	}

	current, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return err
	}
	if _, err := s.check(ctx, admin, current, code, req); err != nil {
		return err
	}

	removed, err := s.store.RemoveAdminPasskey(ctx, admin.ID, factor)
	if err != nil {
		return err
	}
	if !removed {
		return ErrPasskeyUnknown
	}

	s.record(ctx, &admin.ID, admin.Username, "admin.passkey_removed", req, "")

	return nil
}

// PasskeyChallenge offers a session waiting for its second factor a passkey
// sign-in, and answers what navigator.credentials.get needs. Each call
// replaces the ceremony before it.
func (s *Service) PasskeyChallenge(ctx context.Context, token string) (*protocol.PublicKeyCredentialRequestOptions, error) {
	admin, session, state, err := s.Session(ctx, token)
	if err != nil {
		return nil, err
	}
	if state != StateMFA {
		return nil, ErrNoSession
	}

	_, credentials, err := s.passkeys(ctx, admin)
	if err != nil {
		return nil, err
	}
	if len(credentials) == 0 {
		return nil, ErrPasskeyUnknown
	}

	assertion, ceremony, err := s.webauthn.BeginLogin(passkeyAdmin{admin, credentials},
		webauthn.WithUserVerification(protocol.VerificationPreferred))
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(ceremony)
	if err != nil {
		return nil, err
	}
	if err := s.store.OfferAdminPasskey(ctx, session, string(encoded)); err != nil {
		return nil, err
	}

	return &assertion.Response, nil
}

// VerifySignInWithPasskey finishes a sign-in with a passkey's answer to the
// ceremony the session was offered, counted against the lockout as a code
// is, and returns the new session token.
func (s *Service) VerifySignInWithPasskey(ctx context.Context, token string, response []byte, req Request) (*model.Admin, string, error) {
	admin, session, state, err := s.Session(ctx, token)
	if err != nil {
		return nil, "", err
	}
	if state != StateMFA {
		return nil, "", ErrNoSession
	}

	allowed, err := s.store.ReserveAdminLogin(ctx, admin, s.now(), model.MaxFailedLogins, model.LockoutDuration)
	if err != nil {
		return nil, "", err
	}
	if !allowed {
		s.record(ctx, &admin.ID, admin.Username, "admin.mfa_failed", req, "locked after too many attempts")
		return nil, "", ErrInvalidCode
	}

	// The ceremony is spent whatever the answer, so one answer is tried once.
	ceremonyJSON, err := s.store.TakeAdminPasskeyCeremony(ctx, session.ID)
	if err != nil {
		return nil, "", err
	}
	if ceremonyJSON == "" {
		return nil, "", s.failed(ctx, admin, req, "no passkey ceremony offered")
	}

	factor, err := s.passkeyAnswer(ctx, admin, ceremonyJSON, response)
	if err != nil {
		return nil, "", err
	}
	if factor == nil {
		return nil, "", s.failed(ctx, admin, req, "wrong passkey")
	}

	if err := s.store.ClearAdminFailedLogins(ctx, admin); err != nil {
		return nil, "", err
	}

	next, err := s.passMFA(ctx, session)
	if err != nil {
		return nil, "", err
	}
	if err := s.signedIn(ctx, admin, req, "passkey"); err != nil {
		return nil, "", err
	}

	return admin, next, nil
}

// passkeyAnswer is the passkey that answered the ceremony, its counter moved
// on, or nil when the answer is not one of the administrator's.
func (s *Service) passkeyAnswer(ctx context.Context, admin *model.Admin, ceremonyJSON string, response []byte) (*model.MFA, error) {
	var ceremony webauthn.SessionData
	if err := json.Unmarshal([]byte(ceremonyJSON), &ceremony); err != nil {
		return nil, err
	}

	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		return nil, nil
	}

	factors, credentials, err := s.passkeys(ctx, admin)
	if err != nil {
		return nil, err
	}

	credential, err := s.webauthn.ValidateLogin(passkeyAdmin{admin, credentials}, ceremony, parsed)
	if err != nil {
		return nil, nil
	}
	// A counter that went backwards means two copies of the key.
	if credential.Authenticator.CloneWarning {
		s.log.Warn("an administrator's passkey counter went backwards; refusing it", "admin", admin.ID)
		return nil, nil
	}

	id := base64.RawURLEncoding.EncodeToString(credential.ID)
	index := slices.IndexFunc(factors, func(f model.MFA) bool { return f.CredentialID == id })
	if index < 0 {
		return nil, nil
	}

	sealed, err := s.sealJSON(factors[index].ID, credential)
	if err != nil {
		return nil, err
	}
	if err := s.store.UsedAdminPasskey(ctx, factors[index].ID, sealed, s.now()); err != nil {
		return nil, err
	}

	return &factors[index], nil
}

// passkeys are the administrator's confirmed passkeys and their credentials.
func (s *Service) passkeys(ctx context.Context, admin *model.Admin) ([]model.MFA, []webauthn.Credential, error) {
	all, err := s.store.MFAFactors(ctx, admin.ID, model.MFAMethodWebAuthn)
	if err != nil {
		return nil, nil, err
	}

	factors := slices.DeleteFunc(all, func(f model.MFA) bool { return !f.IsConfirmed() })
	credentials := make([]webauthn.Credential, 0, len(factors))
	for _, factor := range factors {
		var credential webauthn.Credential
		if err := s.openJSON(factor.ID, factor.Secret, &credential); err != nil {
			return nil, nil, err
		}
		credentials = append(credentials, credential)
	}

	return factors, credentials, nil
}

func (s *Service) sealJSON(factor uuid.UUID, value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	sealed, err := s.sealer.SealFor(factorSecret(factor), encoded)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *Service) openJSON(factor uuid.UUID, stored string, into any) error {
	sealed, err := base64.StdEncoding.DecodeString(stored)
	if err != nil {
		return err
	}

	plain, _, err := s.sealer.OpenFor(factorSecret(factor), sealed)
	if err != nil {
		return err
	}

	return json.Unmarshal(plain, into)
}
