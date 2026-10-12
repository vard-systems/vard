package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"loginer/internal/model"
	"loginer/internal/totp"
)

// Two-factor sign-in for administrators: a TOTP app plus recovery codes. The
// TOTP secret is sealed because it must be read back; recovery codes are stored
// hashed.

// recoveryCodeCount is how many recovery codes a factor comes with.
const recoveryCodeCount = 10

// The errors of managing a second factor. They are for the administrator to
// read, so they say what to do.
var (
	// ErrInvalidCode is a code that is not right, already used, or expired.
	ErrInvalidCode = errors.New("that code is not right; check the time on your phone and try the newest code")
	// ErrNotEnrolling is confirming when no authenticator is being set up.
	ErrNotEnrolling = errors.New("start setting up an authenticator first")
	// ErrMFAEnabled is starting a set-up without proving the current factor.
	ErrMFAEnabled = errors.New("two-factor sign-in is already on; give a code from your current authenticator to replace it")
	// ErrMFADisabled is managing a factor that is not there.
	ErrMFADisabled = errors.New("two-factor sign-in is not on")
	// ErrMFARequired is turning two-factor sign-in off where it is required.
	ErrMFARequired = errors.New("two-factor sign-in is required for every administrator and cannot be turned off; replace your authenticator instead")
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
}

// Status describes an administrator's second factor.
func (s *Service) Status(ctx context.Context, admin *model.Admin) (MFAStatus, error) {
	status := MFAStatus{Required: s.MFARequired(ctx)}

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

	return status, nil
}

// VerifySignIn finishes a sign-in with a TOTP or recovery code. A wrong one
// counts toward the lockout.
func (s *Service) VerifySignIn(ctx context.Context, token, code string, req Request) (*model.Admin, error) {
	admin, session, state, err := s.Session(ctx, token)
	if err != nil {
		return nil, err
	}
	if state != StateMFA {
		return nil, ErrNoSession
	}

	factor, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return nil, err
	}

	method, err := s.check(ctx, admin, factor, code, req)
	if err != nil {
		return nil, err
	}

	if err := s.store.PassSessionMFA(ctx, session.ID, time.Now().Add(SessionLifetime)); err != nil {
		return nil, err
	}

	if err := s.signedIn(ctx, admin, req, method); err != nil {
		return nil, err
	}

	return admin, nil
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

	sealed, err := s.sealer.SealBytes([]byte(secret))
	if err != nil {
		return nil, err
	}

	factor := model.MFA{
		AdminID: admin.ID,
		Method:  model.MFAMethodTOTP,
		Label:   "Authenticator app",
		Secret:  base64.StdEncoding.EncodeToString(sealed),
	}
	if err := s.store.StartMFA(ctx, &factor); err != nil {
		return nil, err
	}

	return &Enrolment{Secret: secret, URI: totp.URI(s.issuer, admin.Email, secret)}, nil
}

// ConfirmTOTP finishes setup with a code and returns the recovery codes, the
// only time they are in the clear. A session waiting to enrol becomes fully
// signed in.
func (s *Service) ConfirmTOTP(ctx context.Context, admin *model.Admin, token, code string, req Request) ([]string, error) {
	factors, err := s.store.MFAFactors(ctx, admin.ID, model.MFAMethodTOTP)
	if err != nil {
		return nil, err
	}

	var pending *model.MFA
	for i := range factors {
		if !factors[i].IsConfirmed() {
			pending = &factors[i]
			break
		}
	}
	if pending == nil {
		return nil, ErrNotEnrolling
	}

	secret, err := s.secretOf(*pending)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	step, ok := totp.Verify(secret, code, now, 0)
	if !ok {
		return nil, ErrInvalidCode
	}

	codes, hashes, err := s.newRecoveryCodes()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	action := "admin.mfa_enabled"
	if replaced {
		action = "admin.mfa_replaced"
	}
	s.record(ctx, &admin.ID, admin.Username, action, req, "")

	// Every other session ends: one opened with only the password must not
	// become fully signed in now that a factor exists.
	if waiting != nil {
		if _, err := s.store.RevokeOtherSessionsFor(ctx, admin.ID, waiting.ID, now); err != nil {
			return nil, err
		}
	} else if err := s.store.RevokeSessionsFor(ctx, admin.ID, now); err != nil {
		return nil, err
	}

	// A session that was waiting for this is signed in by it.
	if enrolling && !waiting.IsMFAPassed {
		if err := s.store.PassSessionMFA(ctx, waiting.ID, now.Add(SessionLifetime)); err != nil {
			return nil, err
		}
		if err := s.signedIn(ctx, admin, req, "totp"); err != nil {
			return nil, err
		}
	}

	return codes, nil
}

// DisableTOTP turns the second factor off given a code, ending every other
// session. It is refused where a factor is required.
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

	if err := s.store.RemoveMFA(ctx, admin.ID, &session.ID, time.Now()); err != nil {
		return err
	}

	s.record(ctx, &admin.ID, admin.Username, "admin.mfa_disabled", req, "")

	return nil
}

// RegenerateRecoveryCodes replaces an administrator's recovery codes, with a
// code to prove it is them. The old codes stop working.
func (s *Service) RegenerateRecoveryCodes(ctx context.Context, admin *model.Admin, code string, req Request) ([]string, error) {
	factor, err := s.confirmedFactor(ctx, admin)
	if err != nil {
		return nil, err
	}

	if _, err := s.check(ctx, admin, factor, code, req); err != nil {
		return nil, err
	}

	codes, hashes, err := s.newRecoveryCodes()
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

// ResetMFA removes another administrator's factor and signs them out
// everywhere, for a lost phone and lost recovery codes.
func (s *Service) ResetMFA(ctx context.Context, target *model.Admin) error {
	return s.store.RemoveMFA(ctx, target.ID, nil, time.Now())
}

// check accepts a code from the authenticator or a recovery code, once, and
// says which it was. A wrong code counts toward the lockout and is logged.
func (s *Service) check(ctx context.Context, admin *model.Admin, factor *model.MFA, code string, req Request) (string, error) {
	now := time.Now()

	// Reserved before comparing, so parallel guesses are bounded by the
	// lockout.
	allowed, err := s.store.ReserveAdminLogin(ctx, admin, now, MaxFailedLogins, LockoutDuration)
	if err != nil {
		return "", err
	}
	if !allowed {
		s.record(ctx, &admin.ID, admin.Username, "admin.mfa_failed", req, "locked after too many attempts")
		return "", ErrInvalidCode
	}

	if isRecoveryCode(code) {
		ok, err := s.store.ConsumeRecoveryCode(ctx, factor.ID, s.hashRecoveryCode(code))
		if err != nil {
			return "", err
		}
		if !ok {
			// Codes handed out before they were keyed are still good once.
			if ok, err = s.store.ConsumeRecoveryCode(ctx, factor.ID, legacyHashRecoveryCode(code)); err != nil {
				return "", err
			}
		}
		if ok {
			return "recovery_code", s.store.ClearAdminFailedLogins(ctx, admin)
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
	// changes made while signed in do not add up to a lockout.
	return "totp", s.store.ClearAdminFailedLogins(ctx, admin)
}

// failed logs a wrong code. It was counted against the administrator before
// it was compared, in check.
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

	plain, err := s.sealer.OpenBytes(sealed)
	if err != nil {
		return "", err
	}

	return string(plain), nil
}

// Recovery codes look like "k3f9x-q2m7p": ten characters that cannot be
// confused with each other when read off paper, split in two.
const (
	recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	// recoveryCodeLetters is how many of them a code is made of, the dash
	// aside. Ten of this alphabet is a little under fifty bits.
	recoveryCodeLetters = 10
)

func (s *Service) newRecoveryCodes() (codes, hashes []string, err error) {
	for range recoveryCodeCount {
		code, err := newRecoveryCode()
		if err != nil {
			return nil, nil, err
		}

		codes = append(codes, code)
		hashes = append(hashes, s.hashRecoveryCode(code))
	}

	return codes, hashes, nil
}

// newRecoveryCode draws each letter uniformly, rejecting bytes past the last
// whole alphabet rather than taking a biased modulo.
func newRecoveryCode() (string, error) {
	// 248: the last byte value that divides into whole alphabets.
	const whole = 256 - 256%len(recoveryAlphabet)

	letters := make([]byte, 0, recoveryCodeLetters)

	for len(letters) < recoveryCodeLetters {
		var batch [recoveryCodeLetters]byte
		if _, err := rand.Read(batch[:]); err != nil {
			return "", fmt.Errorf("auth: generate recovery code: %w", err)
		}

		for _, v := range batch {
			if int(v) >= whole {
				continue
			}

			letters = append(letters, recoveryAlphabet[int(v)%len(recoveryAlphabet)])
			if len(letters) == recoveryCodeLetters {
				break
			}
		}
	}

	half := recoveryCodeLetters / 2

	return string(letters[:half]) + "-" + string(letters[half:]), nil
}

// normaliseRecoveryCode is a recovery code as it is compared: lower case, with
// the dash and any spaces left out, however it was typed.
func normaliseRecoveryCode(code string) string {
	return strings.NewReplacer("-", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(code)))
}

func isRecoveryCode(code string) bool {
	return len(normaliseRecoveryCode(code)) == recoveryCodeLetters
}

// hashRecoveryCode keys the hash with the server's secret: a code has under
// fifty bits, which a bare SHA-256 from a database dump would give up quickly.
func (s *Service) hashRecoveryCode(code string) string {
	return hex.EncodeToString(s.sealer.Tag(recoveryCodePurpose, []byte(normaliseRecoveryCode(code))))
}

// recoveryCodePurpose keeps the recovery codes' key apart from any other use
// of the secret.
const recoveryCodePurpose = "admin recovery code"

// legacyHashRecoveryCode accepts codes issued before they were keyed, so
// existing paper codes still work.
func legacyHashRecoveryCode(code string) string {
	sum := sha256.Sum256([]byte(normaliseRecoveryCode(code)))
	return hex.EncodeToString(sum[:])
}
