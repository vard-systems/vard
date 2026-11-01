package store

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/cache"
)

func (s *Store) LoginFlows(ctx context.Context) ([]model.LoginFlow, error) {
	var flows []model.LoginFlow
	err := s.db.WithContext(ctx).Order("is_default DESC, name").Find(&flows).Error

	return flows, err
}

func (s *Store) LoginFlow(ctx context.Context, id uuid.UUID) (*model.LoginFlow, error) {
	return cached(ctx, s, cache.LoginFlows, "id:"+id.String(), func() (*model.LoginFlow, error) {
		var flow model.LoginFlow
		if err := s.db.WithContext(ctx).First(&flow, "id = ?", id).Error; err != nil {
			return nil, translate(err)
		}

		return &flow, nil
	})
}

func (s *Store) DefaultLoginFlow(ctx context.Context) (*model.LoginFlow, error) {
	var flow model.LoginFlow
	if s.cache.Get(ctx, cache.LoginFlows, "default", &flow) {
		return &flow, nil
	}

	err := translate(s.db.WithContext(ctx).Order("created_at").First(&flow, "is_default = ?", true).Error)
	switch {
	case err == nil:
		s.cache.Set(ctx, cache.LoginFlows, "default", flow)
		return &flow, nil
	case !errors.Is(err, ErrNotFound):
		return nil, err
	}

	flow = model.DefaultLoginFlow()
	if err := s.db.WithContext(ctx).Create(&flow).Error; err != nil {
		return nil, translate(err)
	}

	s.forget(ctx, cache.LoginFlows)

	return &flow, nil
}

func (s *Store) EffectiveLoginFlow(ctx context.Context, app *model.Application) (*model.LoginFlow, error) {
	if app != nil && app.LoginFlowID != nil {
		flow, err := s.LoginFlow(ctx, *app.LoginFlowID)
		switch {
		case err == nil && flow.IsEnabled:
			return flow, nil
		case err != nil && !errors.Is(err, ErrNotFound):
			return nil, err
		}
	}

	return s.DefaultLoginFlow(ctx)
}

// EnabledLoginFlowOffers reports whether a flow that is on has the step.
func (s *Store) EnabledLoginFlowOffers(ctx context.Context, step model.LoginStep) (bool, error) {
	return cached(ctx, s, cache.LoginFlows, "offers:"+string(step), func() (bool, error) {
		var flows int64
		err := s.db.WithContext(ctx).Model(&model.LoginFlow{}).
			Where("is_enabled AND steps @> ?::jsonb", `["`+string(step)+`"]`).
			Count(&flows).Error

		return flows > 0, err
	})
}

func (s *Store) CreateLoginFlow(ctx context.Context, flow *model.LoginFlow) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(flow).Error; err != nil {
			return translate(err)
		}

		return demoteOtherDefaults(tx, flow)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.LoginFlows)

	return nil
}

func (s *Store) SaveLoginFlow(ctx context.Context, flow *model.LoginFlow) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(flow).Error; err != nil {
			return translate(err)
		}

		return demoteOtherDefaults(tx, flow)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.LoginFlows)

	return nil
}

func demoteOtherDefaults(tx *gorm.DB, flow *model.LoginFlow) error {
	if !flow.IsDefault {
		return nil
	}

	err := tx.Model(&model.LoginFlow{}).
		Where("id <> ? AND is_default = ?", flow.ID, true).
		Update("is_default", false).Error

	return translate(err)
}

var ErrDefaultLoginFlow = errors.New("the default flow cannot be removed")

func (s *Store) DeleteLoginFlow(ctx context.Context, flow *model.LoginFlow) error {
	if flow.IsDefault {
		return ErrDefaultLoginFlow
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.Application{}).
			Where("login_flow_id = ?", flow.ID).
			Update("login_flow_id", nil).Error
		if err != nil {
			return translate(err)
		}

		return translate(tx.Delete(flow).Error)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.LoginFlows, cache.Clients)

	return nil
}

func (s *Store) LoginFlowApplications(ctx context.Context) (map[uuid.UUID]int, error) {
	var rows []struct {
		LoginFlowID uuid.UUID
		Count       int
	}

	err := s.db.WithContext(ctx).
		Model(&model.Application{}).
		Select("login_flow_id, count(*) as count").
		Where("login_flow_id IS NOT NULL").
		Group("login_flow_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		counts[row.LoginFlowID] = row.Count
	}

	return counts, nil
}
