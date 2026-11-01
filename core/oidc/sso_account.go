package oidc

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
)

// The account a single sign-on lands on, and what the enterprise's provider
// keeps up to date on it: the profile, and roles from its groups.

func (s *Service) finishSSO(
	ctx context.Context,
	connection *model.SSOConnection,
	login *model.SSOLogin,
	person *ssoPerson,
	client Client,
) (*SocialResult, error) {
	flow, err := s.flowFor(ctx, login.Request)
	if err != nil {
		return nil, err
	}

	result, err := s.signInThroughSSO(ctx, connection, person, flow, login.Request, client)
	if err != nil {
		return nil, s.ssoFailed(ctx, connection, client, err)
	}

	return &SocialResult{SignIn: result, Request: login.Request, Next: login.Next, Provider: connection.Name}, nil
}

func (s *Service) signInThroughSSO(
	ctx context.Context,
	connection *model.SSOConnection,
	person *ssoPerson,
	flow *model.LoginFlow,
	request string,
	client Client,
) (*SignInResult, error) {
	now := s.now()
	email := strings.ToLower(strings.TrimSpace(person.Email))

	// Check a closed flow first, before any account is created or roles synced.
	if !flow.AllowSignIn {
		return nil, ErrSignInClosed
	}

	switch {
	case person.Subject == "":
		return nil, refused(ErrSSONoEmail, "the provider named nobody: no subject")
	case email == "":
		return nil, refused(ErrSSONoEmail, "no address in %s; the provider sent %s",
			strings.Join(connection.Attribute("email"), " or "), strings.Join(person.Sent, ", "))
	case !connection.OwnsEmail(email):
		return nil, refused(ErrSSODomainMismatch, "%s is outside the connection's domains", email)
	}

	// A connection without domains is trusted with any address but the ones
	// another connection holds, enabled or not: otherwise a partner's
	// provider could sign in as anybody at a company with SSO of its own.
	if len(connection.Domains) == 0 {
		_, domain, _ := strings.Cut(email, "@")
		taken, err := s.store.SSODomainsTaken(ctx, []string{domain}, connection.ID)
		if err != nil {
			return nil, err
		}
		if len(taken) > 0 {
			return nil, refused(ErrSSODomainMismatch, "%s is at a domain another connection holds", email)
		}
	}

	var user *model.User

	identity, err := s.store.SSOIdentity(ctx, connection.ID, person.Subject)
	switch {
	case err == nil:
		if user, err = s.store.User(ctx, identity.UserID); err != nil {
			return nil, err
		}
		if err := s.store.MarkSSOIdentityUsed(ctx, identity, email, now); err != nil {
			return nil, err
		}
	case !errors.Is(err, store.ErrNotFound):
		return nil, err
	default:
		var existing bool
		if user, existing, err = s.ssoAccountFor(ctx, connection, person, email, client); err != nil {
			return nil, err
		}
		if !user.CanSignIn() {
			return nil, s.ssoBlocked(ctx, user, connection, client)
		}

		identity := &model.SSOIdentity{
			UserID: user.ID, ConnectionID: connection.ID, Subject: person.Subject,
			Email: email, LastLoginAt: &now,
		}

		// The provider's word that this is the account's owner is not taken
		// alone the first time: an account with a second factor of its own is
		// asked for it, and only then is the identity linked. Until it is,
		// the provider changes nothing on the account.
		if existing {
			held, err := s.holdSSOLink(ctx, user, flow, identity, request, client)
			if err != nil || held != nil {
				return held, err
			}
		}

		if err := s.linkSSOIdentity(ctx, user, connection.Name, identity, false, client); err != nil {
			return nil, err
		}
	}

	if !user.CanSignIn() {
		return nil, s.ssoBlocked(ctx, user, connection, client)
	}

	if err := s.syncFromSSO(ctx, connection, user, person, client); err != nil {
		return nil, err
	}

	result, err := s.startFederatedSession(ctx, user, flow, request, connection, person, client)
	if err != nil {
		return nil, err
	}

	s.log.Info("a user signed in through single sign-on", "connection", connection.Slug, "user", user.ID)

	return result, nil
}

// ssoAccountFor is the account a person new to the connection lands on, and
// whether it already existed: one with their address, linked, or a new one.
func (s *Service) ssoAccountFor(
	ctx context.Context,
	connection *model.SSOConnection,
	person *ssoPerson,
	email string,
	client Client,
) (*model.User, bool, error) {
	user, err := s.store.UserByEmail(ctx, email)
	switch {
	case err == nil:
		// A connection without domains vouches for addresses nobody gave it,
		// so it never takes an account over, whatever it was saved with.
		if connection.Matching != model.SSOMatchLink || len(connection.Domains) == 0 {
			return nil, false, ErrSSOLinkRefused
		}
		// Never link to an account that has not proved its address, or whoever
		// pre-registered it would keep their password.
		if !user.IsEmailVerified {
			return nil, false, refused(ErrSSOLinkRefused, "the account for %s never verified its address, so it is not linked", email)
		}
		_, domain, _ := strings.Cut(email, "@")
		proved, err := s.store.SSODomainProved(ctx, connection.ID, domain)
		if err != nil {
			return nil, false, err
		}
		if !proved {
			return nil, false, refused(ErrSSOLinkRefused, "the connection has not proved it holds %s, so no existing account there is linked", domain)
		}

		return user, true, nil
	case !errors.Is(err, store.ErrNotFound):
		return nil, false, err
	}

	if !connection.CreateUsers {
		return nil, false, ErrSSONoAccount
	}

	// The provider owns the domain, so the address is as verified as it gets.
	user = &model.User{
		Email:           email,
		IsEmailVerified: true,
		FirstName:       model.Truncate(person.FirstName, 100),
		LastName:        model.Truncate(person.LastName, 100),
		IsActive:        true,
		Data:            map[string]any{},
	}

	roles, err := s.store.DefaultUserRoles(ctx)
	if err != nil {
		return nil, false, err
	}
	user.Roles = roles

	if err := s.store.CreateUser(ctx, user); errors.Is(err, store.ErrDuplicate) {
		return nil, false, ErrSSOExpired
	} else if err != nil {
		return nil, false, err
	}

	s.record(ctx, user, user.Email, "user.registered", client, map[string]any{"provider": connection.Name})

	return user, false, nil
}

// linkSSOIdentity links a provider's identity to an account, and records it
// with whether the account's own second factor was asked for first.
func (s *Service) linkSSOIdentity(ctx context.Context, user *model.User, connection string, identity *model.SSOIdentity, stepped bool, client Client) error {
	if err := s.store.CreateSSOIdentity(ctx, identity); err != nil {
		return err
	}

	s.record(ctx, user, user.Email, "user.identity_connected", client, map[string]any{
		"provider": connection, "second_factor": stepped,
	})

	return nil
}

// holdSSOLink keeps the first link to an account with a second factor
// waiting for it, the identity to link held on the waiting sign-in; nil
// when the account has none.
func (s *Service) holdSSOLink(ctx context.Context, user *model.User, flow *model.LoginFlow, identity *model.SSOIdentity, request string, client Client) (*SignInResult, error) {
	if err := s.admit(ctx, user, flow, request, client); err != nil {
		return nil, err
	}

	return s.hold(ctx, user, request, true, nil, identity)
}

func (s *Service) ssoBlocked(ctx context.Context, user *model.User, connection *model.SSOConnection, client Client) error {
	s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{
		"reason": "inactive", "connection": connection.Name,
	})
	return ErrInvalidCredentials
}

func (s *Service) syncFromSSO(ctx context.Context, connection *model.SSOConnection, user *model.User, person *ssoPerson, client Client) error {
	if connection.SyncProfile {
		first, last := model.Truncate(person.FirstName, 100), model.Truncate(person.LastName, 100)
		if (first != "" && first != user.FirstName) || (last != "" && last != user.LastName) {
			if first != "" {
				user.FirstName = first
			}
			if last != "" {
				user.LastName = last
			}
			if err := s.store.SaveUser(ctx, user); err != nil {
				return err
			}
		}
	}

	if len(connection.RoleMappings) == 0 {
		return nil
	}

	held := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		held = append(held, role.ID.String())
	}

	given := connection.RoleMappings.For(person.Groups)

	var add []model.UserRole
	for _, id := range given {
		if !slices.Contains(held, id.String()) {
			add = append(add, model.UserRole{ID: id})
		}
	}

	var removed []string
	if connection.SyncRoles {
		for _, id := range connection.RoleMappings.Roles() {
			if slices.Contains(held, id.String()) && !slices.Contains(given, id) {
				if err := s.store.RemoveUserRole(ctx, user.ID, id); err != nil {
					return err
				}
				removed = append(removed, id.String())
			}
		}
	}

	if len(add) > 0 {
		existing, err := s.store.UserRolesByID(ctx, rolesIDs(add))
		if err != nil {
			return err
		}
		if err := s.store.AddUserRoles(ctx, user.ID, existing); err != nil {
			return err
		}
	}

	if len(add) > 0 || len(removed) > 0 {
		s.record(ctx, user, user.Email, "user.roles_synced", client, map[string]any{
			"connection": connection.Name, "added": len(add), "removed": len(removed),
		})
	}

	return nil
}

func rolesIDs(roles []model.UserRole) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(roles))
	for _, role := range roles {
		out = append(out, role.ID)
	}

	return out
}
