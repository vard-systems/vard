package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"vard/internal/cache"
	"vard/internal/model"
)

func (s *Store) EnsureSystemAPIs(ctx context.Context) error {
	var admin *model.API

	for _, want := range model.SystemAPIs() {
		api, err := s.ensureSystemAPI(ctx, want)
		if err != nil {
			return err
		}
		if api.System == model.SystemAdminAPI {
			admin = api
		}
	}

	return s.forgetting(ctx, s.ensureAdminCLI(ctx, admin), cache.Grants)
}

func (s *Store) ensureSystemAPI(ctx context.Context, want model.API) (*model.API, error) {
	var found model.API
	err := s.db.WithContext(ctx).Preload("Scopes").First(&found, "system = ?", want.System).Error
	if errors.Is(translate(err), ErrNotFound) {
		if err := s.CreateAPI(ctx, &want); err != nil {
			return nil, err
		}
		return &want, nil
	}
	if err != nil {
		return nil, err
	}

	existing := map[string]model.APIScope{}
	for _, scope := range found.Scopes {
		existing[scope.Name] = scope
	}

	scopes := make([]model.APIScope, 0, len(want.Scopes))
	for _, scope := range want.Scopes {
		if kept, ok := existing[scope.Name]; ok {
			scope.ID = kept.ID
		}
		scopes = append(scopes, scope)
	}

	if found.Identifier == want.Identifier && sameScopes(found.Scopes, scopes) {
		return &found, nil
	}

	found.Identifier = want.Identifier
	found.Scopes = scopes
	if err := s.SaveAPI(ctx, &found); err != nil {
		return nil, err
	}

	return &found, nil
}

func sameScopes(stored, wanted []model.APIScope) bool {
	if len(stored) != len(wanted) {
		return false
	}

	byID := map[uuid.UUID]model.APIScope{}
	for _, scope := range stored {
		byID[scope.ID] = scope
	}
	for _, scope := range wanted {
		have, ok := byID[scope.ID]
		if !ok || have.Name != scope.Name || have.Description != scope.Description || have.IsDefault != scope.IsDefault {
			return false
		}
	}
	return true
}

func (s *Store) ensureAdminCLI(ctx context.Context, admin *model.API) error {
	_, err := s.ApplicationByClientID(ctx, model.AdminCLIClientID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}

	app := &model.Application{
		Name:                 model.AdminCLIClientID,
		Description:          "Calls the admin API as a service. Give it admin API scopes and a secret to use it.",
		Type:                 model.AppM2M,
		ClientID:             model.AdminCLIClientID,
		IsEnabled:            true,
		AccessTokenLifetime:  5 * 60,
		IDTokenLifetime:      model.DefaultIDTokenLifetime,
		RefreshTokenLifetime: model.DefaultRefreshTokenLifetime,
	}
	app.Normalize()

	// A secret nobody has seen: until an administrator rotates it and takes
	// the new one, the application cannot authenticate at all.
	if _, err := app.IssueSecret(time.Now()); err != nil {
		return err
	}

	if err := s.CreateApplication(ctx, app); err != nil {
		return err
	}

	if err := s.AuthorizeApplicationAPI(ctx, app.ID, admin.ID, nil); err != nil {
		return err
	}

	s.forget(ctx, cache.Clients)
	return nil
}
