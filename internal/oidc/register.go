package oidc

import (
	"context"
	"errors"
	"strings"

	"vard/internal/model"
	"vard/internal/store"
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
