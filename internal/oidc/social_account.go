package oidc

import (
	"context"
	"errors"
	"strings"
	"time"

	"vard/internal/model"
	"vard/internal/store"
)

// The account a social sign-in lands on: the one already linked to the
// identity, one with the same verified address, or a new one where the flow
// allows it.

func (s *Service) signInWithIdentity(
	ctx context.Context,
	provider *model.SocialProvider,
	who *socialIdentity,
	request string,
	client Client,
) (*SignInResult, error) {
	now := s.now()
	email := strings.ToLower(strings.TrimSpace(who.Email))

	flow, err := s.flowFor(ctx, request)
	if err != nil {
		return nil, err
	}

	// Closed means closed before any account is made, identity linked or role
	// synced — startSession would refuse at the end, but only after those.
	if !flow.AllowSignIn {
		return nil, ErrSignInClosed
	}

	identity, err := s.store.SocialIdentity(ctx, provider.ID, who.Subject)
	switch {
	case err == nil:
		user, err := s.store.User(ctx, identity.UserID)
		if err != nil {
			return nil, err
		}
		// The account's own address decides SSO enforcement, not the
		// provider's.
		if err := s.socialSSORequired(ctx, user.Email); err != nil {
			return nil, err
		}
		if !user.CanSignIn(now) {
			s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{
				"reason": "inactive", "provider": provider.Name,
			})
			return nil, ErrInvalidCredentials
		}

		if err := s.store.MarkIdentityUsed(ctx, identity, email, now); err != nil {
			return nil, err
		}

		return s.startSocialSession(ctx, user, provider, flow, request, client)
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	if email == "" {
		return nil, ErrSocialNoEmail
	}

	if err := s.socialSSORequired(ctx, email); err != nil {
		return nil, err
	}

	// Linking to an existing account needs the address proved by the provider,
	// a provider trusted to prove it, and the account's own address verified;
	// otherwise whoever pre-registered the address would keep a password to it.
	user, err := s.store.UserByEmail(ctx, email)
	switch {
	case err == nil:
		if !who.EmailVerified || !provider.LinkVerifiedEmails || !user.IsEmailVerified {
			return nil, ErrSocialLinkRefused
		}
		if !user.CanSignIn(now) {
			return nil, ErrInvalidCredentials
		}

		if err := s.connectIdentity(ctx, user, provider, who, email, now); err != nil {
			return nil, err
		}
		s.record(ctx, user, user.Email, "user.identity_connected", client, map[string]any{
			"provider": provider.Name,
		})

		return s.startSocialSession(ctx, user, provider, flow, request, client)
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	}

	if !provider.AllowRegistration || !flow.AllowRegistration {
		return nil, ErrSocialRegistrationClosed
	}
	// An account made for an address nobody proved would be held by whoever
	// controls the provider's account, before the address's owner arrives.
	if !who.EmailVerified {
		return nil, ErrSocialEmailUnverified
	}
	if err := s.socialRegistrationAllowed(ctx, request); err != nil {
		return nil, err
	}

	user = &model.User{
		Email:           email,
		IsEmailVerified: true,
		FirstName:       model.Truncate(who.FirstName, 100),
		LastName:        model.Truncate(who.LastName, 100),
		IsActive:        true,
		Data:            map[string]any{},
	}

	roles, err := s.store.DefaultUserRoles(ctx)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	if err := s.store.CreateUser(ctx, user); errors.Is(err, store.ErrDuplicate) {
		return nil, ErrSocialLinkRefused
	} else if err != nil {
		return nil, err
	}

	if err := s.connectIdentity(ctx, user, provider, who, email, now); err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.registered", client, map[string]any{"provider": provider.Name})

	return s.startSocialSession(ctx, user, provider, flow, request, client)
}

func (s *Service) socialSSORequired(ctx context.Context, email string) error {
	err := s.ssoRequiredFor(ctx, email)

	if _, ok := errors.AsType[*SSORequired](err); ok {
		return ErrSocialSSORequired
	}

	return err
}

func (s *Service) socialRegistrationAllowed(ctx context.Context, request string) error {
	if request == "" {
		return nil
	}

	req, err := s.pending(ctx, request)
	if err != nil {
		return nil
	}

	if !req.Application.AllowRegistration || !req.Application.IsEnabled {
		return ErrSocialRegistrationClosed
	}

	return nil
}

func (s *Service) connectIdentity(
	ctx context.Context,
	user *model.User,
	provider *model.SocialProvider,
	who *socialIdentity,
	email string,
	now time.Time,
) error {
	identity := &model.SocialIdentity{
		UserID:      user.ID,
		ProviderID:  provider.ID,
		Subject:     who.Subject,
		Email:       email,
		LastLoginAt: &now,
	}

	return s.store.CreateSocialIdentity(ctx, identity)
}

func (s *Service) startSocialSession(
	ctx context.Context,
	user *model.User,
	provider *model.SocialProvider,
	flow *model.LoginFlow,
	request string,
	client Client,
) (*SignInResult, error) {
	result, err := s.startSession(ctx, user, flow, request, true, client, "user.login")
	if err != nil {
		return nil, err
	}

	s.log.Info("a user signed in with a social provider",
		"provider", provider.Slug, "user", user.ID)

	return result, nil
}
