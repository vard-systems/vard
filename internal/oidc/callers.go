package oidc

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/google/uuid"

	"loginer/internal/brand"
	"loginer/internal/model"
	"loginer/internal/store"
)

var ErrTokenRefused = errors.New("oidc: the access token is not accepted by this API")

type AdminCaller struct {
	Application *model.Application
	Permissions []string
}

// AdminCaller checks a token presented to the admin API. It must be for the
// admin API and a service's token; a user's token is refused whatever its
// scopes.
func (s *Service) AdminCaller(ctx context.Context, token string) (*AdminCaller, error) {
	claims, app, err := s.caller(ctx, token, brand.AdminAPIIdentifier)
	if err != nil {
		return nil, err
	}
	if claims.Subject != claims.ClientID {
		return nil, ErrTokenRefused
	}

	audience, err := s.store.AudienceFor(ctx, app.ID, brand.AdminAPIIdentifier)
	if err != nil {
		return nil, err
	}
	if !audience.Authorized {
		return nil, ErrTokenRefused
	}

	catalog := model.AdminPermissionNames()
	permissions := []string{}
	for scope := range strings.FieldsSeq(claims.Scope) {
		if slices.Contains(audience.Allowed, scope) && slices.Contains(catalog, scope) {
			permissions = append(permissions, scope)
		}
	}

	return &AdminCaller{Application: app, Permissions: permissions}, nil
}

type AccountCaller struct {
	Session  *Session
	ClientID string
	Scopes   []string
}

func (s *Service) AccountCaller(ctx context.Context, token string) (*AccountCaller, error) {
	claims, app, err := s.caller(ctx, token, brand.AccountAPIIdentifier)
	if err != nil {
		return nil, err
	}

	// Access and scopes are rechecked on every call, so revoking them applies
	// at once.
	audience, err := s.store.AudienceFor(ctx, app.ID, brand.AccountAPIIdentifier)
	if err != nil {
		return nil, err
	}
	if !audience.Authorized {
		return nil, ErrTokenRefused
	}

	id, err := uuid.Parse(claims.Subject)
	if err != nil || claims.Subject == claims.ClientID {
		return nil, ErrTokenRefused
	}

	user, err := s.store.User(ctx, id)
	if errors.Is(err, store.ErrNotFound) {
		return nil, ErrTokenRefused
	}
	if err != nil {
		return nil, err
	}
	// Active, and not unlocked: a token was issued to a signed-in user, and
	// the lock after wrong passwords is on signing in (see SessionFor).
	if !user.IsActive {
		return nil, ErrTokenRefused
	}

	scopes := []string{}
	for scope := range strings.FieldsSeq(claims.Scope) {
		if (scope == model.ScopeAccountRead || scope == model.ScopeAccountWrite) && slices.Contains(audience.Allowed, scope) {
			scopes = append(scopes, scope)
		}
	}

	return &AccountCaller{Session: &Session{User: user}, ClientID: claims.ClientID, Scopes: scopes}, nil
}

func (s *Service) caller(ctx context.Context, token, audience string) (*accessClaims, *model.Application, error) {
	claims, raw, err := s.verifyAccessToken(ctx, token)
	if err != nil || !slices.Contains(claims.Audience, audience) {
		return nil, nil, ErrTokenRefused
	}

	// These APIs do not check DPoP proofs, so a token bound to a key is not
	// taken here as a plain bearer token: that would undo the binding.
	if boundKey(raw) != "" {
		return nil, nil, ErrTokenRefused
	}

	app, err := s.store.ApplicationByClientID(ctx, claims.ClientID)
	if errors.Is(err, store.ErrNotFound) {
		return nil, nil, ErrTokenRefused
	}
	if err != nil {
		return nil, nil, err
	}
	if !app.IsEnabled {
		return nil, nil, ErrTokenRefused
	}

	return claims, app, nil
}
