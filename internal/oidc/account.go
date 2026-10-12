package oidc

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"loginer/internal/model"
	"loginer/internal/store"
)

var ErrWrongPassword = problem("wrong_password", "the current password is wrong")

var ErrNotYours = problem("not_yours", "not the user's")

var ErrPasswordUnchanged = problem("password_unchanged", "the new password is the current one")

var ErrCurrentSession = problem("current_session", "that is the session in use")

type Profile struct {
	FirstName string
	LastName  string
}

func (s *Service) UpdateProfile(ctx context.Context, session *Session, p Profile, client Client) (*model.User, error) {
	user := session.User
	user.FirstName = strings.TrimSpace(p.FirstName)
	user.LastName = strings.TrimSpace(p.LastName)

	// Only the name is written, so a reset, lock or deactivation committed
	// meanwhile is not rolled back.
	if err := s.store.SaveUserProfile(ctx, user); err != nil {
		return nil, err
	}

	s.record(ctx, user, user.Email, "user.profile_updated", client, nil)

	return user, nil
}

func (s *Service) ChangePassword(ctx context.Context, session *Session, current, next string, client Client) error {
	user := session.User
	now := s.now()

	if err := s.checkCurrentPassword(ctx, user, current, client); err != nil {
		return err
	}

	if current == next {
		return ErrPasswordUnchanged
	}

	if err := user.SetPassword(next); errors.Is(err, model.ErrPasswordTooLong) {
		return ErrPasswordTooLong
	} else if err != nil {
		return err
	}
	user.IsPasswordTemporary = false

	if err := s.store.ChangeUserPassword(ctx, user, session.Record.ID, now); err != nil {
		return err
	}

	s.record(ctx, user, user.Email, "user.password_changed_self", client, nil)

	return nil
}

// checkCurrentPassword proves the person at a signed-in session knows the
// password, before a change that would hand them the account. Each guess is
// counted before comparing, like a sign-in, so a session cannot be used to
// find the password past the lockout.
func (s *Service) checkCurrentPassword(ctx context.Context, user *model.User, password string, client Client) error {
	now := s.now()

	allowed, err := s.store.ReserveUserLogin(ctx, user, now, maxFailedLogins, lockoutDuration)
	if err != nil {
		return err
	}
	if !allowed {
		s.record(ctx, user, user.Email, "user.login_blocked", client, map[string]any{"reason": "locked"})
		return ErrWrongPassword
	}

	if user.PasswordHash == "" || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		s.record(ctx, user, user.Email, "user.login_failed", client, map[string]any{"reason": "wrong current password"})
		return ErrWrongPassword
	}

	return s.store.ClearUserFailedLogins(ctx, user, now)
}

// SessionInfo is one browser a user is signed in on.
type SessionInfo struct {
	ID        uuid.UUID `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	SignedIn  time.Time `json:"signed_in_at"`
	ExpiresAt time.Time `json:"expires_at"`
	// Current marks the session the request itself carries.
	Current bool `json:"current"`
}

func (s *Service) Sessions(ctx context.Context, session *Session) ([]SessionInfo, error) {
	records, err := s.store.ActiveUserSessions(ctx, session.User.ID, s.now())
	if err != nil {
		return nil, err
	}

	out := make([]SessionInfo, 0, len(records))
	for _, record := range records {
		out = append(out, SessionInfo{
			ID:        record.ID,
			IP:        record.IP,
			UserAgent: record.UserAgent,
			SignedIn:  record.AuthenticatedAt,
			ExpiresAt: record.ExpiresAt,
			Current:   record.ID == session.Record.ID,
		})
	}

	return out, nil
}

func (s *Service) EndSession(ctx context.Context, session *Session, id uuid.UUID, client Client) error {
	if id == session.Record.ID {
		return ErrCurrentSession
	}

	records, err := s.store.ActiveUserSessions(ctx, session.User.ID, s.now())
	if err != nil {
		return err
	}

	if !slices.ContainsFunc(records, func(r model.UserSession) bool { return r.ID == id }) {
		return ErrNotYours
	}

	if err := s.store.RevokeUserSession(ctx, id, s.now()); err != nil {
		return err
	}

	s.record(ctx, session.User, session.User.Email, "user.session_revoked", client, nil)

	return nil
}

// ConnectedApplication is an application that can still act for the user:
// one holding a refresh token for them.
type ConnectedApplication struct {
	ClientID   string    `json:"client_id"`
	Name       string    `json:"name"`
	LogoURL    string    `json:"logo_url"`
	WebsiteURL string    `json:"website_url"`
	Scopes     []string  `json:"scopes"`
	Authorized time.Time `json:"authorized_at"`
	LastUsed   time.Time `json:"last_used_at"`
}

func (s *Service) ConnectedApplications(ctx context.Context, session *Session) ([]ConnectedApplication, error) {
	grants, err := s.store.UserGrants(ctx, session.User.ID, s.now())
	if err != nil {
		return nil, err
	}

	out := make([]ConnectedApplication, 0, len(grants))
	for _, grant := range grants {
		scopes := []string{}
		for _, scope := range strings.Fields(strings.Join(grant.Scopes, " ")) {
			if !slices.Contains(scopes, scope) {
				scopes = append(scopes, scope)
			}
		}

		out = append(out, ConnectedApplication{
			ClientID:   grant.Application.ClientID,
			Name:       grant.Application.Name,
			LogoURL:    grant.Application.LogoURL,
			WebsiteURL: grant.Application.WebsiteURL,
			Scopes:     scopes,
			Authorized: grant.FirstIssued,
			LastUsed:   grant.LastIssued,
		})
	}

	return out, nil
}

func (s *Service) DisconnectApplication(ctx context.Context, session *Session, clientID string, client Client) error {
	app, err := s.store.ApplicationByClientID(ctx, clientID)
	if errors.Is(err, store.ErrNotFound) {
		return ErrNotYours
	}
	if err != nil {
		return err
	}

	revoked, err := s.store.RevokeUserGrant(ctx, session.User.ID, app.ID, s.now())
	if err != nil {
		return err
	}
	if revoked == 0 {
		return ErrNotYours
	}

	s.record(ctx, session.User, session.User.Email, "user.application_disconnected", client, map[string]any{"application": app.Name})

	return nil
}
