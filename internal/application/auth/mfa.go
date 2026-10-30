package auth

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/domain/recovery"
	"vard/internal/domain/totp"
	"vard/internal/jose"
)

var (
	ErrInvalidCode  = errors.New("that code is not right; check the time on your phone and try the newest code")
	ErrNotEnrolling = errors.New("start setting up an authenticator first")
	ErrMFAEnabled   = errors.New("two-factor sign-in is already on; give a code from your current authenticator to replace it")
	ErrMFADisabled  = errors.New("two-factor sign-in is not on")
	ErrMFARequired  = errors.New("two-factor sign-in is required for every administrator and cannot be turned off; replace your authenticator instead")
)

// Enrolment is an authenticator being set up: what the app needs.
type Enrolment struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// MFAStatus is what an administrator sees about their own second factor.
type MFAStatus struct {
	Enabled           bool       `json:"enabled"`
	Required          bool       `json:"required"`
	ConfirmedAt       *time.Time `json:"confirmed_at"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	RecoveryCodesLeft int        `json:"recovery_codes_left"`
	Passkeys          []Passkey  `json:"passkeys"`
}

func (s *Service) Status(ctx context.Context, admin *model.Admin) (MFAStatus, error) {
	status := MFAStatus{Required: s.MFARequired(ctx), Passkeys: []Passkey{}}

	factor, err := s.confirmedFactor(ctx, admin)
	if errors.Is(err, ErrMFADisabled) {
		return status, nil
	}
	if err != nil {
		return status, err
	}

	status.Enabled = true
	status.ConfirmedAt = factor.ConfirmedAt
	status.LastUsedAt = factor.LastUsedAt
	status.RecoveryCodesLeft = len(factor.RecoveryCodes)

	passkeys, _, err := s.passkeys(ctx, admin)
	if err != nil {
		return status, err
	}
	for _, passkey := range passkeys {
		status.Passkeys = append(status.Passkeys, Passkey{
			ID: passkey.ID.String(), Label: passkey.Label, AddedAt: *passkey.ConfirmedAt, LastUsedAt: passkey.LastUsedAt,
		})
	}

	return status, nil
}

// VerifySignIn finishes a sign-in with its second factor, and returns the
// new session token that replaces the one the password set.
func (s *Service) VerifySignIn(ctx context.Context, token, code string, req Request) (*model.Admin, string, error) {
	admin, session, state, err := s.Session(ctx, token)
	if err != nil {
		return nil, "", err
	}
	if state != StateMFA {
		return nil, "", ErrNoSession
	}

	factor, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return nil, "", err
	}

	method, err := s.check(ctx, admin, factor, code, req)
	if err != nil {
		return nil, "", err
	}

	next, err := s.passMFA(ctx, session)
	if err != nil {
		return nil, "", err
	}

	if err := s.signedIn(ctx, admin, req, method); err != nil {
		return nil, "", err
	}

	return admin, next, nil
}

// BeginTOTP starts authenticator setup. Replacing an active one needs a code
// from it, so the password alone cannot swap it.
func (s *Service) BeginTOTP(ctx context.Context, admin *model.Admin, currentCode string, req Request) (*Enrolment, error) {
	if current, err := s.confirmedFactor(ctx, admin); err == nil {
		if strings.TrimSpace(currentCode) == "" {
			return nil, ErrMFAEnabled
		}
		if _, err := s.check(ctx, admin, current, currentCode, req); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, ErrMFADisabled) {
		return nil, err
	}

	secret, err := totp.NewSecret()
	if err != nil {
		return nil, err
	}

	// The id is the factor's before it is stored, so the seed is sealed to it.
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	sealed, err := s.sealer.SealFor(factorSecret(id), []byte(secret))
	if err != nil {
		return nil, err
	}

	factor := model.MFA{
		AdminID: admin.ID,
		Method:  model.MFAMethodTOTP,
		Label:   "Authenticator app",
		Secret:  base64.StdEncoding.EncodeToString(sealed),
	}
	factor.ID = id
	if err := s.store.StartMFA(ctx, &factor); err != nil {
		return nil, err
	}

	return &Enrolment{Secret: secret, URI: totp.URI(s.issuer, admin.Email, secret)}, nil
}

// ConfirmTOTP turns a pending authenticator on and returns its recovery
// codes. A sign-in that was waiting to enrol is signed in by it, under a new
// session token, returned as `next`; "" otherwise.
func (s *Service) ConfirmTOTP(ctx context.Context, admin *model.Admin, token, code string, req Request) (codes []string, next string, err error) {
	factors, err := s.store.MFAFactors(ctx, admin.ID, model.MFAMethodTOTP)
	if err != nil {
		return nil, "", err
	}

	var pending *model.MFA
	for i := range factors {
		if !factors[i].IsConfirmed() {
			pending = &factors[i]
			break
		}
	}
	if pending == nil {
		return nil, "", ErrNotEnrolling
	}

	secret, err := s.secretOf(*pending)
	if err != nil {
		return nil, "", err
	}

	now := s.now()
	step, ok := totp.Verify(secret, code, now, 0)
	if !ok {
		return nil, "", ErrInvalidCode
	}

	codes, hashes, err := recovery.New(s.sealer, recoveryCodePurpose)
	if err != nil {
		return nil, "", err
	}

	usedAt := time.Unix(step*int64(totp.Period/time.Second), 0)
	replaced := admin.HasMFA()

	// Read before saving the factor: afterwards this session would look like
	// one waiting for a code.
	_, waiting, state, err := s.Session(ctx, token)
	enrolling := err == nil && state == StateEnroll && waiting.AdminID == admin.ID

	pending.ConfirmedAt = &now
	pending.LastUsedAt = &usedAt
	pending.RecoveryCodes = hashes
	if err := s.store.ConfirmMFA(ctx, pending); err != nil {
		return nil, "", err
	}

	action := "admin.mfa_enabled"
	if replaced {
		action = "admin.mfa_replaced"
	}
	s.record(ctx, &admin.ID, admin.Username, action, req, "")

	// Every other session ends: one opened with only the password must not
	// become fully signed in now that a factor exists.
	if waiting != nil {
		if _, err := s.store.RevokeOtherAdminSessionsFor(ctx, admin.ID, waiting.ID, now); err != nil {
			return nil, "", err
		}
	} else if err := s.store.RevokeAdminSessionsFor(ctx, admin.ID, now); err != nil {
		return nil, "", err
	}

	if enrolling && !waiting.IsMFAPassed {
		if next, err = s.passMFA(ctx, waiting); err != nil {
			return nil, "", err
		}
		if err := s.signedIn(ctx, admin, req, "totp"); err != nil {
			return nil, "", err
		}
	}

	return codes, next, nil
}

func (s *Service) DisableTOTP(ctx context.Context, admin *model.Admin, token, code string, req Request) error {
	if s.MFARequired(ctx) {
		return ErrMFARequired
	}

	factor, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return err
	}

	if _, err := s.check(ctx, admin, factor, code, req); err != nil {
		return err
	}

	_, session, _, err := s.Session(ctx, token)
	if err != nil {
		return err
	}

	if err := s.store.RemoveMFA(ctx, admin.ID, &session.ID, s.now()); err != nil {
		return err
	}

	s.record(ctx, &admin.ID, admin.Username, "admin.mfa_disabled", req, "")

	return nil
}

func (s *Service) RegenerateRecoveryCodes(ctx context.Context, admin *model.Admin, code string, req Request) ([]string, error) {
	factor, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return nil, err
	}

	if _, err := s.check(ctx, admin, factor, code, req); err != nil {
		return nil, err
	}

	codes, hashes, err := recovery.New(s.sealer, recoveryCodePurpose)
	if err != nil {
		return nil, err
	}

	factor.RecoveryCodes = hashes
	if err := s.store.SetRecoveryCodes(ctx, factor); err != nil {
		return nil, err
	}

	s.record(ctx, &admin.ID, admin.Username, "admin.recovery_codes_regenerated", req, "")

	return codes, nil
}

func (s *Service) ResetMFA(ctx context.Context, target *model.Admin) error {
	return s.store.RemoveMFA(ctx, target.ID, nil, s.now())
}

func (s *Service) check(ctx context.Context, admin *model.Admin, factor *model.MFA, code string, req Request) (string, error) {
	now := s.now()

	// Reserved before comparing, so parallel guesses are bounded by the
	// throttle.
	allowed, err := s.reserve(ctx, admin)
	if err != nil {
		return "", err
	}
	if !allowed {
		s.record(ctx, &admin.ID, admin.Username, "admin.mfa_failed", req, "throttled after too many attempts")
		return "", ErrInvalidCode
	}

	if recovery.Is(code) {
		ok, err := s.store.ConsumeRecoveryCode(ctx, factor.ID, recovery.Hash(s.sealer, recoveryCodePurpose, code))
		if err != nil {
			return "", err
		}
		if ok {
			return "recovery_code", s.succeeded(ctx, admin)
		}
		return "", s.failed(ctx, admin, req, "wrong recovery code")
	}

	secret, err := s.secretOf(*factor)
	if err != nil {
		return "", err
	}

	step, ok := totp.Verify(secret, code, now, factor.UsedStep(totp.Period))
	if !ok {
		return "", s.failed(ctx, admin, req, "wrong or reused code")
	}

	claimed, err := s.store.ClaimMFAStep(ctx, factor.ID, time.Unix(step*int64(totp.Period/time.Second), 0))
	if err != nil {
		return "", err
	}
	if !claimed {
		return "", s.failed(ctx, admin, req, "code used twice")
	}

	// A right code gives back the attempt it took, so codes asked for by
	// changes made while signed in do not add up to a wait.
	return "totp", s.succeeded(ctx, admin)
}

func (s *Service) failed(ctx context.Context, admin *model.Admin, req Request, reason string) error {
	s.record(ctx, &admin.ID, admin.Username, "admin.mfa_failed", req, reason)

	return ErrInvalidCode
}

func (s *Service) confirmedFactor(ctx context.Context, admin *model.Admin) (*model.MFA, error) {
	factors, err := s.store.MFAFactors(ctx, admin.ID, model.MFAMethodTOTP)
	if err != nil {
		return nil, err
	}

	for i := range factors {
		if factors[i].IsConfirmed() {
			return &factors[i], nil
		}
	}

	return nil, ErrMFADisabled
}

func (s *Service) secretOf(factor model.MFA) (string, error) {
	sealed, err := base64.StdEncoding.DecodeString(factor.Secret)
	if err != nil {
		return "", fmt.Errorf("auth: stored TOTP secret is not base64: %w", err)
	}

	plain, _, err := s.sealer.OpenFor(factorSecret(factor.ID), sealed)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

// factorSecret is where an administrator's factor keeps its sealed secret.
func factorSecret(factor uuid.UUID) jose.Context {
	return model.MFA{Base: model.Base{ID: factor}}.SecretWhere()
}

const recoveryCodePurpose = "admin recovery code"
