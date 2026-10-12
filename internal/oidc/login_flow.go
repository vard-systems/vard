package oidc

import (
	"context"

	"loginer/internal/model"
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
