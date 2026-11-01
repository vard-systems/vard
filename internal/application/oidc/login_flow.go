package oidc

import (
	"context"
	"errors"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
)

func (s *Service) LoginOptions(ctx context.Context, handle string) (model.PublicLoginOptions, error) {
	flow, err := s.flowFor(ctx, handle)
	if err != nil {
		return model.PublicLoginOptions{}, err
	}

	return flow.PublicOptions(), nil
}

func (s *Service) flowFor(ctx context.Context, handle string) (*model.LoginFlow, error) {
	var app *model.Application

	if handle != "" {
		if req, err := s.pending(ctx, handle); err == nil {
			app = req.Application
		}
	}

	return s.store.EffectiveLoginFlow(ctx, app)
}

// flowOf is the login flow a waiting sign-in is held to: its authorization
// request's application's, else the application it names (a device being
// allowed), else the default.
func (s *Service) flowOf(ctx context.Context, pending *model.PendingSignIn) (*model.LoginFlow, error) {
	if pending.Request != "" || pending.ApplicationID == nil {
		return s.flowFor(ctx, pending.Request)
	}

	app, err := s.store.Application(ctx, *pending.ApplicationID)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, ErrSecondFactorExpired
	case err != nil:
		return nil, err
	}

	return s.store.EffectiveLoginFlow(ctx, app)
}
