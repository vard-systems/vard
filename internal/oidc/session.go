package oidc

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"vard/internal/model"
	"vard/internal/store"
)

const MinPasswordLength = 8

type Session struct {
	Record model.UserSession
	User   *model.User
}

type SignInResult struct {
	Session    *Session
	Token      string
	ResetToken string
	Code       *CodeChallenge
	// SecondFactorHandle is set when the sign-in waits for the user's second
	// factor: no session yet, and the handle goes in a cookie.
	SecondFactorHandle string
	Request            string
	Remember           bool
	// KnownDevice is the browser's known-device cookie as this sign-in
	// leaves it, naming this account too.
	KnownDevice string
}

func (s *Session) InBrowser() bool {
	return s.Record.ID != uuid.Nil
}

func (s *Service) SessionFor(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, nil
	}

	record, err := s.store.UserSessionByHash(ctx, model.HashSecret(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	}

	now := s.now()
	if !record.Active(now) {
		return nil, nil
	}

	user, err := s.store.User(ctx, record.UserID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nil
	case err != nil:
		return nil, err
	}

	// The lockout is on password sign-in, not on existing sessions; otherwise
	// anyone could sign a user out with wrong passwords.
	if !user.IsActive {
		return nil, nil
	}

	return &Session{Record: *record, User: user}, nil
}

func (s *Service) SignIn(ctx context.Context, email, password, request string, remember bool, client Client) (*SignInResult, error) {
	// Domains that must use SSO are refused before any password check,
	// revealing nothing about the account.
	if err := s.ssoRequiredFor(ctx, email); err != nil {
		return nil, err
	}

	// Neither does a flow without a password step: it is refused before the
	// password is looked at, for the same reason.
	flow, err := s.flowFor(ctx, request)
	if err != nil {
		return nil, err
	}
	if !flow.Offers(model.StepPassword) {
		return nil, ErrPasswordNotOffered
	}

	// Counted before anything is looked up, against the address typed
	// whether or not anybody has it, so a throttled address tells nothing;
	// a waiting attempt still pays a dummy hash for equal timing.
	keys := s.keysFor(email, client)
	allowed, err := s.reserve(ctx, keys, client)
	if err != nil {
		return nil, err
	}
	if !allowed {
		model.CompareDummyPassword(password)
		s.record(ctx, nil, email, "user.login_blocked", client, map[string]any{"reason": "throttled"})
		return nil, ErrInvalidCredentials
	}

	user, err := s.store.UserByEmail(ctx, email)
	switch {
	case errors.Is(err, store.ErrNotFound):
		model.CompareDummyPassword(password)
		s.record(ctx, nil, email, "user.login_failed", client, map[string]any{"reason": "unknown email"})
		return nil, ErrInvalidCredentials
	case err != nil:
		return nil, err
	}

	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		if user.PasswordHash == "" {
			model.CompareDummyPassword(password)
		}

		s.record(ctx, user, user.Email, "user.login_failed", client, map[string]any{"reason": "wrong password"})

		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "inactive"})

		return nil, ErrInvalidCredentials
	}

	// A right password returns the attempt now, before any emailed code —
	// unless a second factor is still to come. Then only that returns it:
	// otherwise a right password between wrong codes would reset the count,
	// and codes could be guessed for ever.
	secondFactor, err := s.store.HasUserFactor(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	if !secondFactor {
		if err := s.succeeded(ctx, keys); err != nil {
			return nil, err
		}
	}

	if user.IsPasswordTemporary {
		token, err := s.newReset(ctx, user)
		if err != nil {
			return nil, err
		}

		return &SignInResult{ResetToken: token}, nil
	}

	return s.finishSignIn(ctx, user, flow, request, remember, client, "user.login")
}

// finishSignIn ends a password or registration sign-in, holding it for the
// emailed code when the flow asks. Provider sign-ins call startSession
// directly.
func (s *Service) finishSignIn(
	ctx context.Context,
	user *model.User,
	flow *model.LoginFlow,
	request string,
	remember bool,
	client Client,
	action string,
) (*SignInResult, error) {
	if !flow.Offers(model.StepEmailCode) {
		return s.startSession(ctx, user, flow, request, remember, client, action)
	}

	if flow.RequireVerifiedEmail && !user.IsEmailVerified {
		return s.startSession(ctx, user, flow, request, remember, client, action)
	}

	return s.sendLoginCode(ctx, user, request, remember, client)
}

// startSession is where every way of signing in ends: the flow's checks, the
// user's second factor if they have one, then the session.
func (s *Service) startSession(
	ctx context.Context,
	user *model.User,
	flow *model.LoginFlow,
	request string,
	remember bool,
	client Client,
	action string,
) (*SignInResult, error) {
	if err := s.admit(ctx, user, flow, request, client); err != nil {
		return nil, err
	}

	held, err := s.holdForSecondFactor(ctx, user, request, remember)
	if err != nil || held != nil {
		return held, err
	}

	return s.openSession(ctx, user, flow, request, remember, client, action)
}

// startFederatedSession is startSession for single sign-on, whose provider
// answers for the second factor of the people it signs in.
func (s *Service) startFederatedSession(
	ctx context.Context,
	user *model.User,
	flow *model.LoginFlow,
	request string,
	client Client,
) (*SignInResult, error) {
	if err := s.admit(ctx, user, flow, request, client); err != nil {
		return nil, err
	}

	return s.openSession(ctx, user, flow, request, true, client, "user.login")
}

// admit is what the flow asks of everyone, checked here because every way in
// ends here.
func (s *Service) admit(ctx context.Context, user *model.User, flow *model.LoginFlow, request string, client Client) error {
	if !flow.AllowSignIn {
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "sign-in is closed"})

		return ErrSignInClosed
	}

	if flow.RequireVerifiedEmail && !user.IsEmailVerified {
		if err := s.sendVerification(ctx, user, request, client); err != nil {
			return err
		}
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "email not verified"})

		return ErrEmailNotVerified
	}

	return nil
}

func (s *Service) openSession(
	ctx context.Context,
	user *model.User,
	flow *model.LoginFlow,
	request string,
	remember bool,
	client Client,
	action string,
) (*SignInResult, error) {
	now := s.now()

	token, hash, err := model.NewSecret()
	if err != nil {
		return nil, err
	}

	record := model.UserSession{
		TokenHash:       hash,
		UserID:          user.ID,
		AuthenticatedAt: now,
		ExpiresAt:       now.Add(time.Duration(flow.SessionLifetimeHours) * time.Hour),
		IP:              client.IP,
		UserAgent:       model.Truncate(client.UserAgent, 255),
	}
	if err := s.store.CreateUserSession(ctx, &record); err != nil {
		return nil, err
	}

	if err := s.store.MarkUserSignedIn(ctx, user, now); err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, action, client, nil)

	device, err := s.rememberDevice(client.Device, user.Email)
	if err != nil {
		return nil, err
	}

	return &SignInResult{
		Session:     &Session{Record: record, User: user},
		Token:       token,
		Request:     request,
		Remember:    remember && flow.AllowRememberMe,
		KnownDevice: device,
	}, nil
}

func (s *Service) SignOut(ctx context.Context, token string, client Client) error {
	session, err := s.SessionFor(ctx, token)
	if err != nil || session == nil {
		return err
	}

	if err := s.store.RevokeUserSession(ctx, session.Record.ID, s.now()); err != nil {
		return err
	}

	s.record(ctx, session.User, session.User.Email, "user.logout", client, nil)

	return nil
}
