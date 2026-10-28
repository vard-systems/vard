package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"

	"vard/internal/model"
	"vard/internal/recovery"
	"vard/internal/store"
	"vard/internal/totp"
)

// A user's own two-step verification. It is theirs to turn on, from the
// account page, where the flow offers it (the "totp" step): an authenticator
// app, passkeys, or both, with recovery codes for when neither is at hand.
// Once a user has a confirmed factor, every sign-in with a password, an
// emailed code or another account waits for one before its session starts —
// whichever flow it comes through, so turning the step off later does not
// undo anyone's protection. Single sign-on is left to the enterprise's own
// provider, which answers for its people's second factor.

// PageSecondFactor is where a sign-in waiting for its second factor is
// finished.
const PageSecondFactor = "/second-factor"

const (
	MethodAuthenticator = "totp"
	MethodPasskey       = "passkey"
	MethodRecoveryCode  = "recovery_code"

	userRecoveryPurpose = "user recovery code"

	// maxPasskeys is plenty for a person's devices, and keeps the list a
	// sign-in sends small.
	maxPasskeys = 10
)

var (
	ErrSecondFactorInvalid = problem("second_factor_invalid")
	ErrSecondFactorExpired = problem("second_factor_expired")
	ErrTwoStepNotOffered   = problem("two_step_not_offered")
	ErrAuthenticatorExists = problem("authenticator_exists")
	ErrPasskeyLimit        = problem("passkey_limit")
	ErrPasskeyRefused      = problem("passkey_refused")
)

// SecondFactorChallenge is what the second-factor page offers. The handle
// that names the sign-in is never in it: it travels in a cookie.
type SecondFactorChallenge struct {
	Methods   []string  `json:"methods"`
	ExpiresAt time.Time `json:"expires_at"`
	// Passkey is the request for navigator.credentials.get, when the user
	// has a passkey.
	Passkey *protocol.PublicKeyCredentialRequestOptions `json:"passkey,omitempty"`
}

// SecondFactorAnswer is a code (from the authenticator, or a recovery code)
// or a passkey's response, as the browser sent it.
type SecondFactorAnswer struct {
	Code    string
	Passkey []byte
}

// TwoStepStatus is what the account page shows of the user's two-step
// verification.
type TwoStepStatus struct {
	// Offered is whether factors may be added here; one already on can always
	// be removed.
	Offered bool `json:"offered"`
	Enabled bool `json:"enabled"`
	// AsksPassword is whether removing a factor asks for the password: only
	// an account that has one.
	AsksPassword      bool         `json:"asks_password"`
	Authenticator     *FactorInfo  `json:"authenticator"`
	Passkeys          []FactorInfo `json:"passkeys"`
	RecoveryCodesLeft int          `json:"recovery_codes_left"`
}

type FactorInfo struct {
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	AddedAt    time.Time  `json:"added_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// AuthenticatorSetup is an authenticator being set up: what the app needs.
type AuthenticatorSetup struct {
	Secret string `json:"secret"`
	URI    string `json:"uri"`
}

// holdForSecondFactor keeps a sign-in waiting when the user has a second
// factor, and returns nil when they have none.
func (s *Service) holdForSecondFactor(ctx context.Context, user *model.User, request string, remember bool) (*SignInResult, error) {
	return s.hold(ctx, user, request, remember, nil)
}

// hold is holdForSecondFactor with, for a first single sign-on link, the
// identity to link once the factor is answered.
func (s *Service) hold(ctx context.Context, user *model.User, request string, remember bool, link *model.SSOIdentity) (*SignInResult, error) {
	has, err := s.store.HasUserFactor(ctx, user.ID)
	if err != nil || !has {
		return nil, err
	}

	handle, hash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	pending := model.PendingSignIn{
		HandleHash: hash,
		UserID:     user.ID,
		Request:    request,
		RememberMe: remember,
		ExpiresAt:  s.now().Add(model.PendingSignInLifetime),
	}
	if link != nil {
		pending.LinkConnectionID = &link.ConnectionID
		pending.LinkSubject = link.Subject
		pending.LinkEmail = link.Email
	}
	if err := s.store.CreatePendingSignIn(ctx, &pending); err != nil {
		return nil, err
	}

	return &SignInResult{SecondFactorHandle: handle, Request: request, Remember: remember}, nil
}

// SecondFactorFor is what the second-factor page offers for a waiting
// sign-in. Each call starts a fresh passkey ceremony, so the page always
// answers the latest one.
func (s *Service) SecondFactorFor(ctx context.Context, handle string) (*SecondFactorChallenge, error) {
	pending, err := s.waitingSignIn(ctx, handle)
	if err != nil {
		return nil, err
	}

	factors, err := s.confirmedFactors(ctx, pending.UserID)
	if err != nil {
		return nil, err
	}

	challenge := &SecondFactorChallenge{Methods: []string{}, ExpiresAt: pending.ExpiresAt}

	if slices.ContainsFunc(factors, isAuthenticator) {
		challenge.Methods = append(challenge.Methods, MethodAuthenticator)
	}

	credentials, err := s.passkeyCredentials(factors)
	if err != nil {
		return nil, err
	}
	if len(credentials) > 0 {
		assertion, ceremony, err := s.webauthn.BeginLogin(passkeyUser{pending.User, credentials},
			webauthn.WithUserVerification(protocol.VerificationPreferred))
		if err != nil {
			return nil, err
		}
		encoded, err := json.Marshal(ceremony)
		if err != nil {
			return nil, err
		}
		if err := s.store.OfferPasskey(ctx, pending, string(encoded)); err != nil {
			return nil, err
		}

		challenge.Methods = append(challenge.Methods, MethodPasskey)
		challenge.Passkey = &assertion.Response
	}

	challenge.Methods = append(challenge.Methods, MethodRecoveryCode)

	return challenge, nil
}

// SubmitSecondFactor finishes a waiting sign-in with its second factor. Every
// answer counts towards the account's lockout before it is checked, as a
// password does, so guessing codes across many sign-ins is bounded too.
func (s *Service) SubmitSecondFactor(ctx context.Context, handle string, answer SecondFactorAnswer, client Client) (*SignInResult, error) {
	pending, err := s.waitingSignIn(ctx, handle)
	if err != nil {
		return nil, err
	}

	user := pending.User
	now := s.now()
	if !user.CanSignIn(now) {
		return nil, ErrInvalidCredentials
	}

	keys := s.keysFor(user.Email, client)
	allowed, err := s.reserve(ctx, keys, client)
	if err != nil {
		return nil, err
	}
	if !allowed {
		s.record(ctx, user, user.Email, "user.second_factor_failed", client, map[string]any{"reason": "throttled"})
		return nil, ErrInvalidCredentials
	}

	method, err := s.checkSecondFactor(ctx, user, pending, answer)
	if err != nil {
		return nil, err
	}
	if method == "" {
		s.record(ctx, user, user.Email, "user.second_factor_failed", client, nil)
		return nil, ErrSecondFactorInvalid
	}

	used, err := s.store.ConsumePendingSignIn(ctx, pending, now)
	if err != nil {
		return nil, err
	}
	if !used {
		return nil, ErrSecondFactorExpired
	}

	if err := s.succeeded(ctx, keys); err != nil {
		return nil, err
	}

	flow, err := s.flowFor(ctx, pending.Request)
	if err != nil {
		return nil, err
	}
	if err := s.admit(ctx, user, flow, pending.Request, client); err != nil {
		return nil, err
	}

	if err := s.linkHeld(ctx, user, pending, client); err != nil {
		return nil, err
	}

	return s.openSession(ctx, user, flow, pending.Request, pending.RememberMe, client, "user.login")
}

func (s *Service) waitingSignIn(ctx context.Context, handle string) (*model.PendingSignIn, error) {
	if handle == "" {
		return nil, ErrSecondFactorExpired
	}

	pending, err := s.store.PendingSignInByHash(ctx, model.HashSecret(handle))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrSecondFactorExpired
	case err != nil:
		return nil, err
	case !pending.Usable(s.now()) || pending.User == nil:
		return nil, ErrSecondFactorExpired
	}

	return pending, nil
}

// checkSecondFactor says which factor the answer proved, or "" for none.
func (s *Service) checkSecondFactor(ctx context.Context, user *model.User, pending *model.PendingSignIn, answer SecondFactorAnswer) (string, error) {
	now := s.now()

	if len(answer.Passkey) > 0 {
		ok, err := s.checkPasskey(ctx, user, pending, answer.Passkey, now)
		if err != nil || !ok {
			return "", err
		}
		return MethodPasskey, nil
	}

	if recovery.Is(answer.Code) {
		ok, err := s.store.ConsumeUserRecoveryCode(ctx, user.ID, recovery.Hashes(s.sealer, userRecoveryPurpose, answer.Code), now)
		if err != nil || !ok {
			return "", err
		}
		return MethodRecoveryCode, nil
	}

	factors, err := s.confirmedFactors(ctx, user.ID)
	if err != nil {
		return "", err
	}

	for _, factor := range factors {
		if !isAuthenticator(factor) {
			continue
		}

		seed, err := s.unseal(factor.ID, factor.Secret)
		if err != nil {
			return "", err
		}

		step, ok := totp.Verify(string(seed), answer.Code, now, factor.UsedStep(totp.Period))
		if !ok {
			continue
		}

		// Of two sign-ins racing with one code, only one takes its step.
		claimed, err := s.store.ClaimUserFactorStep(ctx, factor.ID, time.Unix(step*int64(totp.Period/time.Second), 0))
		if err != nil || !claimed {
			return "", err
		}
		return MethodAuthenticator, nil
	}

	return "", nil
}

// linkHeld links the single sign-on identity a waiting sign-in held, now that
// the account's own second factor has been answered.
func (s *Service) linkHeld(ctx context.Context, user *model.User, pending *model.PendingSignIn, client Client) error {
	if pending.LinkConnectionID == nil {
		return nil
	}

	connection, err := s.store.SSOConnection(ctx, *pending.LinkConnectionID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrSecondFactorExpired
	}
	if err != nil {
		return err
	}

	now := s.now()
	identity := &model.SSOIdentity{
		UserID: user.ID, ConnectionID: connection.ID, Subject: pending.LinkSubject,
		Email: pending.LinkEmail, LastLoginAt: &now,
	}

	// Two waiting links racing: the first made it, which is what this one
	// would have made.
	if err := s.linkSSOIdentity(ctx, user, connection.Name, identity, true, client); err != nil && !errors.Is(err, store.ErrDuplicate) {
		return err
	}

	return nil
}
