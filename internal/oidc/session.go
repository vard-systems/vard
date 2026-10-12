package oidc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"loginer/i18n"
	"loginer/internal/mail"
	"loginer/internal/model"
	"loginer/internal/store"
)

// MinPasswordLength is the shortest password a user may choose, the same as
// an administrator may give them.
const MinPasswordLength = 8

// Session is a signed-in user in one browser.
type Session struct {
	Record model.UserSession
	User   *model.User
}

// SignInResult is what signing in or registering produced.
type SignInResult struct {
	Session *Session
	// Token is the session cookie's value. It is the only copy.
	Token string
	// ResetToken replaces the session when the user signed in with a temporary
	// password and must choose a new one.
	ResetToken string
	// Code replaces the session when the flow requires the emailed code
	// (SubmitLoginCode).
	Code *CodeChallenge
	// Request is the sign-in this belongs to, for callers that did not start
	// it.
	Request string
	// Remember makes the cookie outlive the browser window; the session's own
	// lifetime is the flow's.
	Remember bool
}

// dummyHash is compared against for an unknown address, so it takes as long to
// refuse as a wrong password.
var dummyHash = sync.OnceValue(func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("not a password anyone has"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("oidc: make the dummy hash: %v", err))
	}
	return hash
})

// InBrowser reports whether the session is a browser's, signed in at the
// sign-in pages, rather than a user an application's access token names.
func (s *Session) InBrowser() bool {
	return s.Record.ID != uuid.Nil
}

// SessionFor returns the session a cookie carries, or nil when it carries none
// that still signs a user in.
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

// SignIn checks a user's address and password and starts a session, for the
// sign-in under way that `request` names, if any.
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
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		s.record(ctx, nil, email, "user.login_failed", client, map[string]any{"reason": "unknown email"})
		return nil, ErrInvalidCredentials
	case err != nil:
		return nil, err
	}

	// Reserve the attempt before comparing, so parallel guesses cannot pass the
	// lock; a locked account still pays a dummy hash for equal timing.
	allowed, err := s.store.ReserveUserLogin(ctx, user, now, maxFailedLogins, lockoutDuration)
	if err != nil {
		return nil, err
	}
	if !allowed {
		_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "locked"})
		return nil, ErrInvalidCredentials
	}

	// A user an administrator made without a password has none to match.
	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		if user.PasswordHash == "" {
			_ = bcrypt.CompareHashAndPassword(dummyHash(), []byte(password))
		}

		s.record(ctx, user, user.Email, "user.login_failed", client, map[string]any{"reason": "wrong password"})

		return nil, ErrInvalidCredentials
	}

	// Inactive is refused here; the lock is already handled by the reservation.
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

	// A flow that requires a verified address still requires it first: there
	// is no point emailing a code to an address the flow will not take.
	if flow.RequireVerifiedEmail && !user.IsEmailVerified {
		return s.startSession(ctx, user, flow, request, remember, client, action)
	}

	return s.sendLoginCode(ctx, user, request, remember, client)
}

// startSession signs a user in under the flow's rules (verified address,
// session length). Every way in ends here.
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
		UserAgent:       truncate(client.UserAgent, 255),
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

// SignOut ends the session a cookie carries. Signing out twice is not an error.
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

// Registration is what someone creating an account types.
type Registration struct {
	// Request is the sign-in under way the account is made for. Registering
	// is offered per application, so there is no registering without one.
	Request   string
	Email     string
	Password  string
	FirstName string
	LastName  string
	// AcceptedTerms says the person agreed to the application's terms and
	// privacy policy. It is required when the application links either.
	AcceptedTerms bool
	// Remember is the "stay signed in" box, as on the sign-in page.
	Remember bool
}

// Register makes an account for a sign-in under way, and signs it in.
func (s *Service) Register(ctx context.Context, r Registration, client Client) (*SignInResult, error) {
	// Accounts at a domain its identity provider owns are made by signing in
	// through it.
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

	// The flow decides whether the installation takes new accounts at all, and
	// a flow without a password step only creates them through its providers.
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
		if _, err := field.Normalise(nil); err != nil {
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

	// The verification email does not hold the account back
	// (RequireVerifiedEmail does), so a send failure is logged and the link can
	// be resent.
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
	// A domain its identity provider owns keeps its passwords there.
	if err := s.ssoRequiredFor(ctx, email); err != nil {
		return err
	}

	// A flow without password reset sends nothing, with the same answer.
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
		// Throttled: no new link, same quiet success.
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
		// The request that asked may be long gone by the time this sends.
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

// CheckReset reports whether a reset link still works, so the page can say so
// before someone types a new password into it.
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

// ResetPassword sets a new password from a reset link and ends every session
// and refresh token.
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

// sendVerification sends a verification link while the person waits, so a send
// failure is reported.
func (s *Service) sendVerification(ctx context.Context, user *model.User, request string, client Client) error {
	return s.mailVerification(ctx, user, "", request, client)
}

// mailVerification sends one link: to the current address, or with `newEmail`
// to that address as a pending change.
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

	// The link goes where the address is being proved: the account's own, or
	// the one somebody is moving to.
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

// VerifyEmail uses a verification link: the address is the user's, and every
// other link sent to it stops working.
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
		// Somebody took the address between the link being sent and used.
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

// truncate cuts a value to `max` bytes on a rune boundary and drops invalid
// UTF-8, which Postgres would refuse and fail the whole sign-in over (e.g. a
// long Cyrillic name or User-Agent).
func truncate(value string, max int) string {
	value = strings.ToValidUTF8(value, "")
	if len(value) <= max {
		return value
	}

	// Back up off a continuation byte to the start of the rune it belongs to.
	for max > 0 && !utf8.RuneStart(value[max]) {
		max--
	}

	return value[:max]
}
