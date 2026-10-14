package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"loginer/i18n"
	"loginer/internal/mail"
	"loginer/internal/model"
	"loginer/internal/store"
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
	Request    string
	Remember   bool
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
	now := s.now()

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

	user, err := s.store.UserByEmail(ctx, email)
	switch {
	case errors.Is(err, store.ErrNotFound):
		model.CompareDummyPassword(password)
		s.record(ctx, nil, email, "user.login_failed", client, map[string]any{"reason": "unknown email"})
		return nil, ErrInvalidCredentials
	case err != nil:
		return nil, err
	}

	// Reserve the attempt before comparing, so parallel guesses cannot pass the
	// lock; a locked account still pays a dummy hash for equal timing.
	allowed, err := s.store.ReserveUserLogin(ctx, user, now, model.MaxFailedLogins, model.LockoutDuration)
	if err != nil {
		return nil, err
	}
	if !allowed {
		model.CompareDummyPassword(password)
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "locked"})
		return nil, ErrInvalidCredentials
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

	// A right password returns the attempt now, before any emailed code.
	if err := s.store.ClearUserFailedLogins(ctx, user, now); err != nil {
		return nil, err
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

func (s *Service) startSession(
	ctx context.Context,
	user *model.User,
	flow *model.LoginFlow,
	request string,
	remember bool,
	client Client,
	action string,
) (*SignInResult, error) {
	now := s.now()

	// A closed flow is checked here because every way in ends here.
	if !flow.AllowSignIn {
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "sign-in is closed"})

		return nil, ErrSignInClosed
	}

	if flow.RequireVerifiedEmail && !user.IsEmailVerified {
		if err := s.sendVerification(ctx, user, request, client); err != nil {
			return nil, err
		}
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "email not verified"})

		return nil, ErrEmailNotVerified
	}

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

	return &SignInResult{
		Session:  &Session{Record: record, User: user},
		Token:    token,
		Request:  request,
		Remember: remember && flow.AllowRememberMe,
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

type Registration struct {
	Request       string
	Email         string
	Password      string
	FirstName     string
	LastName      string
	AcceptedTerms bool
	Remember      bool
}

func (s *Service) Register(ctx context.Context, r Registration, client Client) (*SignInResult, error) {
	if err := s.ssoRequiredFor(ctx, r.Email); err != nil {
		return nil, err
	}

	req, err := s.pending(ctx, r.Request)
	if err != nil {
		return nil, err
	}

	app := req.Application
	if !app.AllowRegistration || !app.IsEnabled {
		return nil, ErrRegistrationClosed
	}

	flow, err := s.store.EffectiveLoginFlow(ctx, app)
	if err != nil {
		return nil, err
	}
	if !flow.AllowRegistration || !flow.Offers(model.StepPassword) {
		return nil, ErrRegistrationClosed
	}

	if (app.TermsURL != "" || app.PrivacyURL != "") && !r.AcceptedTerms {
		return nil, ErrTermsRequired
	}

	// Required additional fields cannot be filled in on a sign-in page, so
	// an installation that has them cannot take registrations.
	fields, err := s.store.UserFields(ctx)
	if err != nil {
		return nil, err
	}
	for _, field := range fields {
		if _, err := field.Normalize(nil); err != nil {
			return nil, ErrDetailsRequired
		}
	}

	user := &model.User{
		Email:     strings.ToLower(strings.TrimSpace(r.Email)),
		FirstName: strings.TrimSpace(r.FirstName),
		LastName:  strings.TrimSpace(r.LastName),
		IsActive:  true,
		Data:      map[string]any{},
	}

	if err := user.SetPassword(r.Password); errors.Is(err, model.ErrPasswordTooLong) {
		return nil, ErrPasswordTooLong
	} else if err != nil {
		return nil, err
	}

	roles, err := s.store.DefaultUserRoles(ctx)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	if err := s.store.CreateUser(ctx, user); errors.Is(err, store.ErrDuplicate) {
		return nil, ErrEmailTaken
	} else if err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.registered", client, map[string]any{"application": app.Name})

	if flow.VerifyEmailOnRegister && !user.IsEmailVerified {
		if err := s.sendVerification(ctx, user, r.Request, client); err != nil {
			s.log.Error("sending a verification email failed", "error", err, "user", user.ID)
		}
	}

	return s.finishSignIn(ctx, user, flow, r.Request, r.Remember, client, "user.login")
}

// ForgotPassword sends a reset link if the address has an account that may sign
// in, in the page's language. It answers the same either way and sends in the
// background, so timing cannot reveal which addresses exist.
func (s *Service) ForgotPassword(ctx context.Context, email, request, language string, client Client) error {
	if err := s.ssoRequiredFor(ctx, email); err != nil {
		return err
	}

	var app *model.Application
	if pending, err := s.pending(ctx, request); err == nil {
		app = pending.Application
	}

	flow, err := s.store.EffectiveLoginFlow(ctx, app)
	if err != nil {
		return err
	}
	if !flow.AllowPasswordReset || !flow.Offers(model.StepPassword) {
		return nil
	}

	user, err := s.store.UserByEmail(ctx, strings.TrimSpace(email))
	if errors.Is(err, store.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	if !user.IsActive {
		return nil
	}

	token, hash, err := model.NewSecret()
	if err != nil {
		return err
	}
	reset := &model.PasswordReset{
		TokenHash: hash,
		UserID:    user.ID,
		ExpiresAt: s.now().Add(model.PasswordResetLifetime),
	}
	created, err := s.store.CreatePasswordResetThrottled(ctx, reset, s.now())
	if err != nil {
		return err
	}
	if !created {
		return nil
	}

	text := s.textIn(ctx, language)

	name := text["email.reset.your_account"]
	if pending, err := s.pending(ctx, request); err == nil {
		name = pending.Application.Name
	} else {
		request = ""
	}

	link := withQuery(s.accountURL+PageReset, url.Values{"token": {token}, "request": {request}})
	params := map[string]any{
		"app":     name,
		"email":   user.Email,
		"link":    link,
		"minutes": int(model.PasswordResetLifetime.Minutes()),
	}

	msg := mail.Message{
		To:      user.Email,
		Subject: i18n.Fill(text["email.reset.subject"], params),
		Body:    i18n.Fill(text["email.reset.body"], params),
	}

	s.record(ctx, user, user.Email, "user.password_reset_requested", client, nil)

	go func() {
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()

		if err := s.mail.Send(sendCtx, msg); err != nil {
			s.log.Error("sending a password reset email failed", "error", err)
		}
	}()

	return nil
}

func (s *Service) newReset(ctx context.Context, user *model.User) (string, error) {
	token, hash, err := model.NewSecret()
	if err != nil {
		return "", err
	}

	err = s.store.CreatePasswordReset(ctx, &model.PasswordReset{
		TokenHash: hash,
		UserID:    user.ID,
		ExpiresAt: s.now().Add(model.PasswordResetLifetime),
	})

	return token, err
}

func (s *Service) CheckReset(ctx context.Context, token string) error {
	_, err := s.reset(ctx, token)
	return err
}

func (s *Service) reset(ctx context.Context, token string) (*model.PasswordReset, error) {
	if token == "" {
		return nil, ErrResetInvalid
	}

	reset, err := s.store.PasswordResetByHash(ctx, model.HashSecret(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrResetInvalid
	case err != nil:
		return nil, err
	}

	if !reset.Usable(s.now()) {
		return nil, ErrResetInvalid
	}

	return reset, nil
}

func (s *Service) ResetPassword(ctx context.Context, token, password string, client Client) error {
	reset, err := s.reset(ctx, token)
	if err != nil {
		return err
	}

	user, err := s.store.User(ctx, reset.UserID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrResetInvalid
	}
	if err != nil {
		return err
	}

	if err := user.SetPassword(password); errors.Is(err, model.ErrPasswordTooLong) {
		return ErrPasswordTooLong
	} else if err != nil {
		return err
	}

	if err := s.store.ResetPassword(ctx, reset, user, s.now()); errors.Is(err, store.ErrAlreadyUsed) {
		return ErrResetInvalid
	} else if err != nil {
		return err
	}

	s.record(ctx, user, user.Email, "user.password_reset", client, nil)

	return nil
}

func (s *Service) sendVerification(ctx context.Context, user *model.User, request string, client Client) error {
	return s.mailVerification(ctx, user, "", request, client)
}

func (s *Service) mailVerification(
	ctx context.Context,
	user *model.User,
	newEmail, request string,
	client Client,
) error {
	token, hash, err := model.NewSecret()
	if err != nil {
		return err
	}

	err = s.store.CreateEmailVerification(ctx, &model.EmailVerification{
		TokenHash: hash,
		UserID:    user.ID,
		NewEmail:  newEmail,
		ExpiresAt: s.now().Add(model.EmailVerificationLifetime),
	})
	if err != nil {
		return err
	}

	text := s.textIn(ctx, client.Language)

	name := text["email.reset.your_account"]
	if pending, err := s.pending(ctx, request); err == nil {
		name = pending.Application.Name
	} else {
		request = ""
	}

	to := user.Email
	if newEmail != "" {
		to = newEmail
	}

	params := map[string]any{
		"app":   name,
		"email": to,
		"link":  withQuery(s.accountURL+PageVerify, url.Values{"token": {token}, "request": {request}}),
		"hours": int(model.EmailVerificationLifetime.Hours()),
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = s.mail.Send(sendCtx, mail.Message{
		To:      to,
		Subject: i18n.Fill(text["email.verify.subject"], params),
		Body:    i18n.Fill(text["email.verify.body"], params),
	})
	if err != nil {
		return fmt.Errorf("send the verification email: %w", err)
	}

	s.record(ctx, user, to, "user.email_verification_sent", client, nil)

	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string, client Client) error {
	if token == "" {
		return ErrVerificationInvalid
	}

	verification, err := s.store.EmailVerificationByHash(ctx, model.HashSecret(token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		return ErrVerificationInvalid
	case err != nil:
		return err
	case !verification.Usable(s.now()):
		return ErrVerificationInvalid
	}

	// A change of address may not move an account off a domain that must use
	// SSO, even with an older link.
	if verification.IsChange() {
		owner, err := s.store.User(ctx, verification.UserID)
		if err != nil {
			return err
		}
		if err := s.ssoRequiredFor(ctx, owner.Email); err != nil {
			return err
		}
	}

	err = s.store.VerifyEmail(ctx, verification, s.now())
	switch {
	case errors.Is(err, store.ErrAlreadyUsed):
		return ErrVerificationInvalid
	case errors.Is(err, store.ErrDuplicate):
		return ErrEmailTaken
	case err != nil:
		return err
	}

	user, err := s.store.User(ctx, verification.UserID)
	if err != nil {
		return err
	}

	action := "user.email_verified"
	if verification.IsChange() {
		action = "user.email_changed"
	}
	s.record(ctx, user, user.Email, action, client, nil)

	return nil
}

// RequestEmailChange emails a link to the new address; the account moves only
// when it is used. It answers the same whether or not the address is taken, so
// it cannot be used to discover accounts.
//
// A new address plus a password reset is the whole account, so it takes the
// current password, and a browser session: an application's access token
// may not move the account away from its owner.
func (s *Service) RequestEmailChange(ctx context.Context, session *Session, email, password string, client Client) error {
	flow, err := s.flowFor(ctx, "")
	if err != nil {
		return err
	}
	if !flow.AllowEmailChange || !session.InBrowser() {
		return ErrEmailChangeNotOffered
	}

	// An account on an SSO-only domain cannot move to another address here;
	// that plus a password reset would bypass SSO.
	if err := s.ssoRequiredFor(ctx, session.User.Email); err != nil {
		return err
	}

	if err := s.checkCurrentPassword(ctx, session.User, password, client); err != nil {
		return err
	}

	email = model.NormalizeEmail(email)
	if email == "" || email == session.User.Email {
		return nil
	}

	if _, err := s.store.UserByEmail(ctx, email); err == nil {
		s.record(ctx, session.User, email, "user.email_change_requested", client, map[string]any{"sent": false})

		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}

	if err := s.mailVerification(ctx, session.User, email, "", client); err != nil {
		return err
	}

	s.record(ctx, session.User, email, "user.email_change_requested", client, map[string]any{"sent": true})

	return nil
}
