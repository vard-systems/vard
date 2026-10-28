package store

import (
	"context"
	"slices"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/cache"
	"vard/internal/model"
)

func (s *Store) MFAFactors(ctx context.Context, admin uuid.UUID, method model.MFAMethod) ([]model.MFA, error) {
	var factors []model.MFA
	err := s.db.WithContext(ctx).
		Where("admin_id = ? AND method = ?", admin, method).
		Order("created_at DESC").
		Find(&factors).Error

	return factors, err
}

// StartMFA stores a new, unconfirmed factor, removing any other unconfirmed one
// of the same method: only the enrolment under way can be confirmed.
func (s *Store) StartMFA(ctx context.Context, factor *model.MFA) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.
			Where("admin_id = ? AND method = ? AND confirmed_at IS NULL", factor.AdminID, factor.Method).
			Delete(&model.MFA{}).Error
		if err != nil {
			return err
		}

		return translate(tx.Omit(clause.Associations).Create(factor).Error)
	})
	if err == nil {
		s.forget(ctx, cache.Admins)
	}

	return err
}

func (s *Store) ConfirmMFA(ctx context.Context, factor *model.MFA) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.
			Where("admin_id = ? AND method = ? AND id <> ?", factor.AdminID, factor.Method, factor.ID).
			Delete(&model.MFA{}).Error
		if err != nil {
			return err
		}

		return tx.Omit(clause.Associations).Save(factor).Error
	})
	if err == nil {
		s.forget(ctx, cache.Admins)
	}

	return err
}

// ClaimMFAStep records the step a code was accepted for; only one of two
// concurrent claims succeeds.
func (s *Store) ClaimMFAStep(ctx context.Context, factor uuid.UUID, at time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Model(&model.MFA{}).
		Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", factor, at).
		Update("last_used_at", at)

	return result.RowsAffected == 1, result.Error
}

// ConsumeRecoveryCode removes one recovery code under a row lock, so it works
// only once even when presented twice. It leaves LastUsedAt alone: that column
// is the last accepted TOTP step, not a timestamp.
func (s *Store) ConsumeRecoveryCode(ctx context.Context, factor uuid.UUID, hashes ...string) (bool, error) {
	consumed := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.MFA
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", factor).Error; err != nil {
			return translate(err)
		}

		index := slices.IndexFunc(row.RecoveryCodes, func(stored string) bool { return slices.Contains(hashes, stored) })
		if index < 0 {
			return nil
		}

		consumed = true
		// Through the struct, with the column named, so the codes go through
		// their JSON serializer; a plain column update would not.
		return tx.Model(&row).Select("RecoveryCodes").Updates(&model.MFA{
			RecoveryCodes: slices.Delete(slices.Clone(row.RecoveryCodes), index, index+1),
		}).Error
	})

	return consumed, err
}

func (s *Store) SetRecoveryCodes(ctx context.Context, factor *model.MFA) error {
	return s.db.WithContext(ctx).Model(factor).Select("RecoveryCodes").Updates(&model.MFA{RecoveryCodes: factor.RecoveryCodes}).Error
}

func (s *Store) RemoveMFA(ctx context.Context, admin uuid.UUID, keep *uuid.UUID, at time.Time) error {
	var ended []model.AdminSession

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_id = ?", admin).Delete(&model.MFA{}).Error; err != nil {
			return err
		}

		condition, args := "admin_id = ?", []any{admin}
		if keep != nil {
			condition, args = "admin_id = ? AND id <> ?", []any{admin, *keep}
		}

		var err error
		ended, err = revokeAdminSessionsIn(tx, at, condition, args...)
		return err
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.Admins)
	s.putAdminSessions(ctx, ended...)

	return nil
}

// StartAdminPasskey keeps a passkey being added, its ceremony in Secret,
// removing any other passkey left half-added. The confirmed ones stay: an
// administrator may have several.
func (s *Store) StartAdminPasskey(ctx context.Context, factor *model.MFA) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("admin_id = ? AND method = ? AND confirmed_at IS NULL", factor.AdminID, model.MFAMethodWebAuthn).
			Delete(&model.MFA{}).Error
		if err != nil {
			return err
		}

		return translate(tx.Omit(clause.Associations).Create(factor).Error)
	})
}

// ConfirmAdminPasskey keeps the credential the browser made in place of the
// ceremony, and turns the passkey on.
func (s *Store) ConfirmAdminPasskey(ctx context.Context, factor *model.MFA) error {
	err := s.db.WithContext(ctx).Model(&model.MFA{}).
		Where("id = ? AND confirmed_at IS NULL", factor.ID).
		Updates(map[string]any{
			"secret": factor.Secret, "credential_id": factor.CredentialID,
			"label": factor.Label, "confirmed_at": factor.ConfirmedAt,
		}).Error
	if err == nil {
		s.forget(ctx, cache.Admins)
	}

	return err
}

// UsedAdminPasskey keeps a passkey's credential as its last sign-in left it,
// its counter moved on.
func (s *Store) UsedAdminPasskey(ctx context.Context, factor uuid.UUID, sealed string, at time.Time) error {
	return s.db.WithContext(ctx).Model(&model.MFA{}).
		Where("id = ?", factor).
		Updates(map[string]any{"secret": sealed, "last_used_at": at}).Error
}

// RemoveAdminPasskey removes one of an administrator's passkeys, and says
// whether there was one.
func (s *Store) RemoveAdminPasskey(ctx context.Context, admin, factor uuid.UUID) (bool, error) {
	result := s.db.WithContext(ctx).
		Where("id = ? AND admin_id = ? AND method = ?", factor, admin, model.MFAMethodWebAuthn).
		Delete(&model.MFA{})
	if result.Error == nil {
		s.forget(ctx, cache.Admins)
	}

	return result.RowsAffected > 0, result.Error
}

// OfferAdminPasskey keeps the passkey ceremony offered to a session waiting
// for its second factor, replacing any earlier one; TakeAdminPasskeyCeremony
// takes it.
func (s *Store) OfferAdminPasskey(ctx context.Context, session *model.AdminSession, ceremony string) error {
	return s.db.WithContext(ctx).Model(&model.AdminSession{}).
		Where("id = ? AND revoked_at IS NULL", session.ID).
		Update("passkey_ceremony", ceremony).Error
}

// TakeAdminPasskeyCeremony takes the passkey ceremony a session was offered,
// once: the first answer spends it, right or wrong. "" when there is none.
// It is read from the database, never the cache, which does not keep it.
func (s *Store) TakeAdminPasskeyCeremony(ctx context.Context, session uuid.UUID) (string, error) {
	var taken []string
	err := s.db.WithContext(ctx).Raw(`
		UPDATE admin_sessions SET passkey_ceremony = ''
		FROM (SELECT id, passkey_ceremony FROM admin_sessions WHERE id = ? FOR UPDATE) offered
		WHERE admin_sessions.id = offered.id AND offered.passkey_ceremony <> '' AND admin_sessions.revoked_at IS NULL
		RETURNING offered.passkey_ceremony`, session).Scan(&taken).Error
	if err != nil || len(taken) == 0 {
		return "", err
	}

	return taken[0], nil
}
