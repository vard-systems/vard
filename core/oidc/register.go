package oidc

import (
	"context"
	"errors"
	"strings"
	"time"

	"vard/core/mail"
	"vard/core/model"
	"vard/core/store"
)

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

	if err := s.refuseBreached(ctx, r.Password); err != nil {
		return nil, err
	}
	if err := user.SetPassword(r.Password); errors.Is(err, model.ErrPasswordTooLong) {
		return nil, ErrPasswordTooLong
	} else if err != nil {
		return nil, err
	}

	// The form's checkbox shows the application's own links, or the
	// organization's when it has none; ticked over the organization's, it is
	// the Agreements step already passed.
	if r.AcceptedTerms && app.TermsURL == "" && app.PrivacyURL == "" && flow.Offers(model.StepTerms) {
		now := s.now()
		user.AgreementsAcceptedAt = &now
	}

	roles, err := s.store.DefaultUserRoles(ctx)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	if err := s.store.CreateUser(ctx, user); errors.Is(err, store.ErrDuplicate) {
		return nil, s.registeredAgain(ctx, user.Email, app, flow, client)
	} else if err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.registered", client, map[string]any{"application": app.Name})

	// A flow that requires a verified address answers every registration
	// "check your email", a taken address's too (registeredAgain); the mail
	// goes after the answer in both, so neither takes longer than the other.
	if flow.RequireVerifiedEmail {
		if !flow.AllowSignIn {
			return nil, ErrSignInClosed
		}
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "email not verified"})
		s.afterAnswer(func(ctx context.Context) {
			if err := s.sendVerification(ctx, user, r.Request, client); err != nil {
				s.log.Error("sending a verification email failed", "error", err, "user", user.ID)
			}
		})
		return nil, ErrEmailNotVerified
	}

	if flow.VerifyEmailOnRegister && !user.IsEmailVerified {
		if err := s.sendVerification(ctx, user, r.Request, client); err != nil {
			s.log.Error("sending a verification email failed", "error", err, "user", user.ID)
		}
	}

	return s.finishSignIn(ctx, user, flow, r.Request, r.Remember, []string{model.AMRPassword}, client, "user.login")
}

// registeredAgain answers a registration for an address that already has an
// account. Where the flow tells a fresh account to confirm its address before
// anything else, this one is told the same, in the same time, and the
// address's owner is mailed a notice instead of a link — at most once every
// model.RegisteredAgainNoticeEvery, so the form cannot be used to flood an
// inbox: the answer then says nothing about which addresses have accounts. A
// flow that signs a fresh account in at once cannot hide it — the stranger
// would be signed in, or would see they were not — so there it says the
// address is taken.
func (s *Service) registeredAgain(ctx context.Context, email string, app *model.Application, flow *model.LoginFlow, client Client) error {
	// What a fresh account is answered in the same flow (Register).
	switch {
	case !flow.RequireVerifiedEmail:
		return ErrEmailTaken
	case !flow.AllowSignIn:
		return ErrSignInClosed
	}

	// A lookup and a write, as the fresh account's record costs; the rest
	// waits for the answer to be gone.
	owner, err := s.store.UserByEmail(ctx, email)
	switch {
	case errors.Is(err, store.ErrNotFound):
		// Deleted between the insert and now: nobody to tell.
		return ErrEmailNotVerified
	case err != nil:
		return err
	}
	s.record(ctx, owner, owner.Email, "user.registration_repeated", client, map[string]any{"application": app.Name})

	s.afterAnswer(func(ctx context.Context) {
		due, err := s.store.ClaimOnceEvery(ctx, "notice:"+s.accountKey(email), s.now(), model.RegisteredAgainNoticeEvery)
		if err != nil {
			s.log.Error("checking the repeated-registration notice's limit failed", "error", err, "user", owner.ID)
			return
		}
		if due {
			s.mailRegisteredAgain(ctx, owner, app, client)
		}
	})

	return ErrEmailNotVerified
}

// afterAnswer runs work a request must not wait on, or be timed by: what one
// address costs that another does not, done once the answer is on its way.
func (s *Service) afterAnswer(work func(ctx context.Context)) {
	background := context.WithoutCancel(context.Background())
	go func() {
		ctx, cancel := context.WithTimeout(background, 30*time.Second)
		defer cancel()
		work(ctx)
	}()
}

// mailRegisteredAgain tells an account's owner that somebody tried to make
// another account with their address. A failure is only logged: the answer
// to the stranger must not change with it.
func (s *Service) mailRegisteredAgain(ctx context.Context, owner *model.User, app *model.Application, client Client) {
	text := s.textIn(ctx, client.Language)

	params := map[string]any{
		"app":   app.Name,
		"email": owner.Email,
		"ip":    client.IP,
	}

	sendCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err := s.mail.Send(sendCtx, mail.Message{
		To:      owner.Email,
		Subject: text.Localize("email.registered_again.subject", params),
		Body:    text.Localize("email.registered_again.body", params),
	})
	if err != nil {
		s.log.Warn("sending the repeated-registration notice failed", "user", owner.ID, "error", err)
	}
}
