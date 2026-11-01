package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/domain/model"
)

// Consent is what a user has allowed an application; ErrNotFound if nothing.
func (s *Store) Consent(ctx context.Context, user, application uuid.UUID) (*model.Consent, error) {
	var consent model.Consent
	err := s.db.WithContext(ctx).First(&consent, "user_id = ? AND application_id = ?", user, application).Error
	if err != nil {
		return nil, translate(err)
	}

	return &consent, nil
}

// AllowScopes adds scopes to what a user has allowed an application. An
// existing row is locked while it changes, so consent pages answered at once
// add to each other rather than one undoing the other.
func (s *Store) AllowScopes(ctx context.Context, user, application uuid.UUID, scopes []string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		consent := model.Consent{UserID: user, ApplicationID: application, Scopes: []string{}}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&consent, "user_id = ? AND application_id = ?", user, application).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			consent.Allow(scopes)
			return tx.Omit(clause.Associations).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(&consent).Error
		case err != nil:
			return err
		}

		consent.Allow(scopes)

		return tx.Model(&consent).Select("scopes").Updates(&consent).Error
	})
}

// ForgetConsent removes what a user allowed an application, so the next
// sign-in asks again. It reports whether there was anything to forget.
func (s *Store) ForgetConsent(ctx context.Context, user, application uuid.UUID) (bool, error) {
	result := s.db.WithContext(ctx).
		Where("user_id = ? AND application_id = ?", user, application).
		Delete(&model.Consent{})

	return result.RowsAffected > 0, result.Error
}

// UserConsents is everything a user has allowed, newest first, with the
// applications.
func (s *Store) UserConsents(ctx context.Context, user uuid.UUID) ([]model.Consent, error) {
	var consents []model.Consent
	err := s.db.WithContext(ctx).
		Preload("Application").
		Where("user_id = ?", user).
		Order("updated_at DESC").
		Find(&consents).Error

	return consents, err
}

// AcceptAgreements records that a user accepted the organization's terms and
// privacy policy.
func (s *Store) AcceptAgreements(ctx context.Context, user *model.User, at time.Time) error {
	if err := s.db.WithContext(ctx).Model(user).Update("agreements_accepted_at", at).Error; err != nil {
		return err
	}
	user.AgreementsAcceptedAt = &at

	return nil
}
