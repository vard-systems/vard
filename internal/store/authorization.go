package store

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"

	"loginer/internal/cache"
	"loginer/internal/model"
)

type Audience struct {
	API        *model.API
	Authorized bool
	Allowed    []string
}

func (s *Store) AudienceFor(ctx context.Context, applicationID uuid.UUID, identifier string) (Audience, error) {
	return cached(ctx, s, cache.Grants, "audience:"+applicationID.String()+":"+identifier, func() (Audience, error) {
		return s.audienceFor(ctx, applicationID, identifier)
	})
}

func (s *Store) audienceFor(ctx context.Context, applicationID uuid.UUID, identifier string) (Audience, error) {
	api, err := s.APIByIdentifier(ctx, identifier)
	if err != nil {
		return Audience{}, err
	}

	var authorized int64
	err = s.db.WithContext(ctx).Model(&model.ApplicationAPI{}).
		Where("application_id = ? AND api_id = ?", applicationID, api.ID).Count(&authorized).Error
	if err != nil {
		return Audience{}, err
	}

	var allowed []uuid.UUID
	err = s.db.WithContext(ctx).Model(&model.ApplicationAPIScope{}).
		Joins("JOIN api_scopes ON api_scopes.id = application_api_scopes.api_scope_id").
		Where("application_api_scopes.application_id = ? AND api_scopes.api_id = ?", applicationID, api.ID).
		Pluck("application_api_scopes.api_scope_id", &allowed).Error
	if err != nil {
		return Audience{}, err
	}

	out := Audience{API: api, Authorized: authorized > 0, Allowed: []string{}}
	for _, scope := range api.Scopes {
		if slices.Contains(allowed, scope.ID) {
			out.Allowed = append(out.Allowed, scope.Name)
		}
	}

	return out, nil
}

func (s *Store) CreateAuthorizationRequest(ctx context.Context, req *model.AuthorizationRequest) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(req).Error)
}

func (s *Store) AuthorizationRequestByHandle(ctx context.Context, hash string) (*model.AuthorizationRequest, error) {
	var req model.AuthorizationRequest
	err := s.db.WithContext(ctx).Preload("Application").First(&req, "handle_hash = ?", hash).Error
	if err != nil {
		return nil, translate(err)
	}

	return &req, nil
}

// CompleteAuthorizationRequest marks a sign-in finished; only the first call
// succeeds, so a handle yields one code.
func (s *Store) CompleteAuthorizationRequest(ctx context.Context, req *model.AuthorizationRequest, at time.Time) error {
	result := s.db.WithContext(ctx).Model(&model.AuthorizationRequest{}).
		Where("id = ? AND completed_at IS NULL", req.ID).
		Update("completed_at", at)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAlreadyUsed
	}

	req.CompletedAt = &at

	return nil
}

func (s *Store) CreateAuthorizationCode(ctx context.Context, code *model.AuthorizationCode) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(code).Error)
}

// ClaimAuthorizationCode marks this application's code used and returns it. A
// concurrent second claim, or a code belonging to another application, gets
// ErrAlreadyUsed with the code, which is never written to, so a client that saw
// someone else's code cannot burn it.
func (s *Store) ClaimAuthorizationCode(ctx context.Context, hash string, application uuid.UUID, at time.Time) (*model.AuthorizationCode, error) {
	var codes []model.AuthorizationCode
	err := s.db.WithContext(ctx).Model(&codes).
		Clauses(clause.Returning{}).
		Where("code_hash = ? AND application_id = ? AND used_at IS NULL", hash, application).
		Update("used_at", at).Error
	if err != nil {
		return nil, err
	}
	if len(codes) == 1 {
		return &codes[0], nil
	}

	var used model.AuthorizationCode
	if err := s.db.WithContext(ctx).First(&used, "code_hash = ?", hash).Error; err != nil {
		return nil, translate(err)
	}

	return &used, ErrAlreadyUsed
}
