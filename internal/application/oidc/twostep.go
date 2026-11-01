package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"slices"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/domain/recovery"
	"vard/internal/domain/totp"
	"vard/internal/infrastructure/store"
)

// A user's own two-step verification. It is theirs to turn on, from the
// account page, where a flow offers it (the "totp" step): an authenticator
// app, passkeys, or both, with recovery codes for when neither is at hand. A
// flow that requires it has the people without one set one up as they sign
// in, on the second-factor page. Once a user has a confirmed factor, every
// sign-in with a password, an emailed code or another account waits for one
// before its session starts — whichever flow it comes through, so turning
// the step off later does not undo anyone's protection. Single sign-on is left to the enterprise's own
// provider, which answers for its people's second factor.

// PageSecondFactor is where a sign-in waiting for its second factor is
// finished.
const PageSecondFactor = "/second-factor"

// factorAMR is what each second factor proves (model.AMR). A passkey here is
// the device alone: the ceremony only prefers the person verified.
var factorAMR = map[string]string{
	MethodAuthenticator: model.AMROneTimeCode,
	MethodRecoveryCode:  model.AMROneTimeCode,
	MethodPasskey:       model.AMRHardwareKey,
}

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
	// ErrSecondFactorRequired refuses a code for an application whose flow
	// requires a second factor the session has not passed.
	ErrSecondFactorRequired = problem("second_factor_required")
)

// SecondFactorChallenge is what the second-factor page offers. The handle
// that names the sign-in is never in it: it travels in a cookie.
type SecondFactorChallenge struct {
	Methods   []string  `json:"methods"`
	ExpiresAt time.Time `json:"expires_at"`
	// StepUp is a signed-in person asked for a second factor in the session
	// they have, for an application that requires one.
	StepUp bool `json:"step_up"`
	// Enrol is a person without a second factor signing in to an application
	// that requires one: Methods are what they may set up, and setting one up
	// finishes the sign-in.
	Enrol bool `json:"enrol"`
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
func (s *Service) holdForSecondFactor(ctx context.Context, user *model.User, request string, remember bool, methods []string) (*SignInResult, error) {
	return s.hold(ctx, user, request, remember, methods, nil)
}

// hold is holdForSecondFactor with, for a first single sign-on link, the
// identity to link once the factor is answered.
func (s *Service) hold(ctx context.Context, user *model.User, request string, remember bool, methods []string, link *model.SSOIdentity) (*SignInResult, error) {
	has, err := s.store.HasUserFactor(ctx, user.ID)
	if err != nil || !has {
		return nil, err
	}

	return s.keepWaiting(ctx, user, request, remember, methods, link)
}

// holdForEnrolment keeps a sign-in waiting while a person without a second
// factor sets one up, for a flow that requires one. Only at a proved
// address: otherwise the verification email goes first, since whoever
// registered an address nobody proved may not be its owner.
func (s *Service) holdForEnrolment(ctx context.Context, user *model.User, request string, remember bool, methods []string, client Client) (*SignInResult, error) {
	if !user.IsEmailVerified {
		if err := s.sendVerification(ctx, user, request, client); err != nil {
			return nil, err
		}
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "email not verified"})

		return nil, ErrEmailNotVerified
	}

	return s.keepWaiting(ctx, user, request, remember, methods, nil)
}

func (s *Service) keepWaiting(ctx context.Context, user *model.User, request string, remember bool, methods []string, link *model.SSOIdentity) (*SignInResult, error) {
	handle, hash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	pending := model.PendingSignIn{
		HandleHash: hash,
		UserID:     user.ID,
		Request:    request,
		RememberMe: remember,
		Methods:    methods,
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

// holdStepUp sends a signed-in browser to pass a second factor in the
// session it has, or to set one up there when the account has none, for an
// application whose flow requires one. held gets the handle for the
// browser's cookie. An account without a factor at an unproved address signs
// in again instead, which sends the verification email first.
func (s *Service) holdStepUp(ctx context.Context, request string, session *Session, held *string) (string, error) {
	has, err := s.store.HasUserFactor(ctx, session.User.ID)
	if err != nil {
		return "", err
	}
	if !has && !session.User.IsEmailVerified {
		return withQuery(s.accountURL+PageLogin, url.Values{"request": {request}}), nil
	}

	handle, err := s.waitForStepUp(ctx, session, request, nil)
	if err != nil {
		return "", err
	}
	*held = handle

	return withQuery(s.accountURL+PageSecondFactor, url.Values{"request": {request}}), nil
}

// waitForStepUp keeps a step-up of the session waiting, for an authorization
// request or, with none, the application named, and returns its handle.
func (s *Service) waitForStepUp(ctx context.Context, session *Session, request string, app *uuid.UUID) (string, error) {
	handle, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	pending := model.PendingSignIn{
		HandleHash:      hash,
		UserID:          session.User.ID,
		Request:         request,
		Methods:         session.Record.Methods,
		StepUpSessionID: &session.Record.ID,
		ApplicationID:   app,
		ExpiresAt:       s.now().Add(model.PendingSignInLifetime),
	}
	if err := s.store.CreatePendingSignIn(ctx, &pending); err != nil {
		return "", err
	}

	return handle, nil
}

// LeaveSignIn gives up a waiting sign-in ("Not now") and says where the
// browser goes: back to the application with access_denied when it is for an
// authorization request, else the account page. The waiting sign-in and the
// request are each taken by one conditional UPDATE, so leaving cannot be
// repeated, nor race a second factor answered or set up at the same time.
func (s *Service) LeaveSignIn(ctx context.Context, handle string, current *Session, client Client) (string, error) {
	pending, err := s.signInFor(ctx, handle, current)
	if err != nil {
		return "", err
	}

	used, err := s.store.ConsumePendingSignIn(ctx, pending, s.now())
	if err != nil {
		return "", err
	}
	if !used {
		return "", ErrSecondFactorExpired
	}

	if pending.Request == "" {
		return s.accountURL + "/", nil
	}

	// A request that has run out has nobody waiting for its answer.
	req, err := s.pending(ctx, pending.Request)
	if errors.Is(err, ErrRequestExpired) {
		return s.accountURL + "/", nil
	}
	if err != nil {
		return "", err
	}
	if err := s.store.CompleteAuthorizationRequest(ctx, req, s.now()); errors.Is(err, store.ErrAlreadyUsed) {
		return "", ErrSecondFactorExpired
	} else if err != nil {
		return "", err
	}

	s.record(ctx, pending.User, pending.User.Email, "user.second_factor_declined", client,
		map[string]any{"application": req.Application.Name})

	return s.redirectError(req.RedirectURI, req.State, ErrAccessDenied, "the person did not continue with two-step verification"), nil
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

	challenge := &SecondFactorChallenge{Methods: []string{}, ExpiresAt: pending.ExpiresAt, StepUp: pending.StepUpSessionID != nil}

	if len(factors) == 0 {
		challenge.Enrol = true
		challenge.Methods = append(challenge.Methods, MethodAuthenticator, MethodPasskey)
		return challenge, nil
	}

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
// answer counts towards the account's throttle before it is checked, as a
// password does, so guessing codes across many sign-ins is bounded too.
// current is the session the browser's cookie names, nil for none: a step-up
// is finished only in the session it was started for.
func (s *Service) SubmitSecondFactor(ctx context.Context, handle string, current *Session, answer SecondFactorAnswer, client Client) (*SignInResult, error) {
	pending, err := s.signInFor(ctx, handle, current)
	if err != nil {
		return nil, err
	}
	stepUp := pending.StepUpSessionID != nil

	user := pending.User
	now := s.now()
	if !user.CanSignIn() {
		return nil, ErrInvalidCredentials
	}

	keys := s.keysFor(user.Email, client)
	allowed, err := s.reserve(ctx, keys)
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

	flow, err := s.flowOf(ctx, pending)
	if err != nil {
		return nil, err
	}
	if err := s.admit(ctx, user, flow, pending.Request, client); err != nil {
		return nil, err
	}

	if stepUp {
		return s.stepUp(ctx, current, pending.Request, factorAMR[method], method, client)
	}

	if err := s.linkHeld(ctx, user, pending, client); err != nil {
		return nil, err
	}

	methods := model.WithSecondFactor(pending.Methods, factorAMR[method])
	return s.openSession(ctx, user, flow, pending.Request, pending.RememberMe, methods, client, "user.login")
}

// stepUp adds a second factor to the session the browser has, under a new
// token: a cookie seen before the factor is worth nothing after it. The
// session keeps its id, so nothing issued in it ends, and its expiry, so a
// step-up never makes it last longer. The new cookie is kept as the old one
// was: past the browser's closing only if the session was "stay signed in".
func (s *Service) stepUp(ctx context.Context, current *Session, request, proved, method string, client Client) (*SignInResult, error) {
	token, hash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	record := current.Record
	stepped, err := s.store.StepUpUserSession(ctx, &record, hash, model.WithSecondFactor(record.Methods, proved), s.now())
	if err != nil {
		return nil, err
	}
	if !stepped {
		return nil, ErrSecondFactorExpired
	}

	s.record(ctx, current.User, current.User.Email, "user.step_up", client, map[string]any{"method": method})

	return &SignInResult{Session: &Session{Record: record, User: current.User}, Token: token, Request: request, Remember: record.Remember}, nil
}

// signInFor is the waiting sign-in the handle names, as the browser that
// holds current may finish it. A step-up's cookie alone upgrades nothing: the
// browser must still hold the session it was for, as the same person.
func (s *Service) signInFor(ctx context.Context, handle string, current *Session) (*model.PendingSignIn, error) {
	pending, err := s.waitingSignIn(ctx, handle)
	if err != nil {
		return nil, err
	}

	if pending.StepUpSessionID != nil &&
		(current == nil || current.Record.ID != *pending.StepUpSessionID || current.User.ID != pending.UserID) {
		return nil, ErrSecondFactorExpired
	}

	return pending, nil
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
		ok, err := s.store.ConsumeUserRecoveryCode(ctx, user.ID, recovery.Hash(s.sealer, userRecoveryPurpose, answer.Code), now)
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
