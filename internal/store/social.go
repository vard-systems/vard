package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/cache"
	"vard/internal/model"
)

func (s *Store) SocialProviders(ctx context.Context, enabledOnly bool) ([]model.SocialProvider, error) {
	query := s.db.WithContext(ctx).Order("position, created_at")
	if enabledOnly {
		query = query.Where("is_enabled = ?", true)
	}

	var providers []model.SocialProvider
	err := query.Find(&providers).Error

	return providers, err
}

type SocialButton struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// SocialButtons returns enabled providers as buttons, through the cache.
// Buttons, not providers, so sealed secrets never reach the cache.
func (s *Store) SocialButtons(ctx context.Context) ([]SocialButton, error) {
	return cached(ctx, s, cache.SocialButtons, "enabled", func() ([]SocialButton, error) {
		providers, err := s.SocialProviders(ctx, true)
		if err != nil {
			return nil, err
		}

		buttons := make([]SocialButton, 0, len(providers))
		for _, provider := range providers {
			buttons = append(buttons, SocialButton{Slug: provider.Slug, Name: provider.Name, Kind: string(provider.Kind)})
		}

		return buttons, nil
	})
}

func (s *Store) SocialProvider(ctx context.Context, id uuid.UUID) (*model.SocialProvider, error) {
	var provider model.SocialProvider
	if err := s.db.WithContext(ctx).First(&provider, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &provider, nil
}

func (s *Store) SocialProviderBySlug(ctx context.Context, slug string) (*model.SocialProvider, error) {
	var provider model.SocialProvider
	if err := s.db.WithContext(ctx).First(&provider, "slug = ?", slug).Error; err != nil {
		return nil, translate(err)
	}

	return &provider, nil
}

func (s *Store) CreateSocialProvider(ctx context.Context, provider *model.SocialProvider) error {
	if err := translate(s.db.WithContext(ctx).Create(provider).Error); err != nil {
		return err
	}

	s.forget(ctx, cache.SocialButtons)

	return nil
}

func (s *Store) SaveSocialProvider(ctx context.Context, provider *model.SocialProvider) error {
	if err := translate(s.db.WithContext(ctx).Save(provider).Error); err != nil {
		return err
	}

	s.forget(ctx, cache.SocialButtons)

	return nil
}

func (s *Store) DeleteSocialProvider(ctx context.Context, provider *model.SocialProvider) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("provider_id = ?", provider.ID).Delete(&model.SocialIdentity{}).Error
		if err != nil {
			return err
		}

		return tx.Delete(provider).Error
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.SocialButtons)

	return nil
}

func (s *Store) NextSocialPosition(ctx context.Context) int {
	var last model.SocialProvider
	if err := s.db.WithContext(ctx).Order("position DESC").First(&last).Error; err != nil {
		return 1
	}

	return last.Position + 1
}

func (s *Store) SocialIdentityCounts(ctx context.Context) (map[uuid.UUID]int64, error) {
	var rows []struct {
		ProviderID uuid.UUID
		Count      int64
	}

	err := s.db.WithContext(ctx).
		Model(&model.SocialIdentity{}).
		Select("provider_id, count(*) AS count").
		Group("provider_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		counts[row.ProviderID] = row.Count
	}

	return counts, nil
}

func (s *Store) SocialIdentity(ctx context.Context, providerID uuid.UUID, subject string) (*model.SocialIdentity, error) {
	var identity model.SocialIdentity
	err := s.db.WithContext(ctx).
		First(&identity, "provider_id = ? AND subject = ?", providerID, subject).Error
	if err != nil {
		return nil, translate(err)
	}

	return &identity, nil
}

func (s *Store) SocialIdentities(ctx context.Context, userID uuid.UUID) ([]model.SocialIdentity, error) {
	var identities []model.SocialIdentity
	err := s.db.WithContext(ctx).
		Preload("Provider").
		Where("user_id = ?", userID).
		Order("created_at").
		Find(&identities).Error

	return identities, err
}

func (s *Store) CreateSocialIdentity(ctx context.Context, identity *model.SocialIdentity) error {
	return translate(s.db.WithContext(ctx).Create(identity).Error)
}

func (s *Store) MarkIdentityUsed(ctx context.Context, identity *model.SocialIdentity, email string, at time.Time) error {
	identity.LastLoginAt = &at
	identity.Email = email

	return translate(s.db.WithContext(ctx).
		Model(identity).
		Updates(map[string]any{"last_login_at": at, "email": email}).Error)
}

func (s *Store) DeleteSocialIdentity(ctx context.Context, identity *model.SocialIdentity) error {
	return s.db.WithContext(ctx).Delete(identity).Error
}

func (s *Store) CreateSocialLogin(ctx context.Context, login *model.SocialLogin) error {
	return translate(s.db.WithContext(ctx).Create(login).Error)
}

// TakeSocialLogin returns and deletes the sign-in a state belongs to, so an
// answer is accepted once. Missing or expired is ErrNotFound.
func (s *Store) TakeSocialLogin(ctx context.Context, stateHash string, now time.Time) (*model.SocialLogin, error) {
	var login model.SocialLogin

	err := s.db.WithContext(ctx).
		Clauses(clause.Returning{}).
		Where("state_hash = ? AND expires_at > ?", stateHash, now).
		Delete(&login).Error
	if err != nil {
		return nil, translate(err)
	}

	if login.ID == uuid.Nil {
		return nil, ErrNotFound
	}

	return &login, nil
}

func (s *Store) SocialAccountsFor(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]model.SocialAccount, error) {
	accounts := map[uuid.UUID][]model.SocialAccount{}
	if len(userIDs) == 0 {
		return accounts, nil
	}

	var rows []struct {
		ID          uuid.UUID
		UserID      uuid.UUID
		Provider    string
		Slug        string
		Kind        model.SocialKind
		Email       string
		ConnectedAt time.Time
		LastLoginAt *time.Time
	}

	err := s.db.WithContext(ctx).
		Table("social_identities AS i").
		Select(`i.id, i.user_id, p.name AS provider, p.slug, p.kind,
			i.email, i.created_at AS connected_at, i.last_login_at`).
		Joins("JOIN social_providers p ON p.id = i.provider_id").
		Where("i.user_id IN ?", userIDs).
		Order("p.position, i.created_at").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		accounts[row.UserID] = append(accounts[row.UserID], model.SocialAccount{
			ID:          row.ID,
			Provider:    row.Provider,
			Slug:        row.Slug,
			Kind:        row.Kind,
			Email:       row.Email,
			ConnectedAt: row.ConnectedAt,
			LastLoginAt: row.LastLoginAt,
		})
	}

	return accounts, nil
}

func (s *Store) SocialIdentityByID(ctx context.Context, id uuid.UUID) (*model.SocialIdentity, error) {
	var identity model.SocialIdentity
	if err := s.db.WithContext(ctx).First(&identity, "id = ?", id).Error; err != nil {
		return nil, translate(err)
	}

	return &identity, nil
}
