package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/internal/model"
)

// CreateDeviceAuthorization stores a device's sign-in. ErrDuplicate when its
// user code is already one waiting: the caller draws another.
func (s *Store) CreateDeviceAuthorization(ctx context.Context, device *model.DeviceAuthorization) error {
	return translate(s.db.WithContext(ctx).Omit(clause.Associations).Create(device).Error)
}

// DeviceByCode is a device's sign-in by its device code, with the application.
func (s *Store) DeviceByCode(ctx context.Context, hash string) (*model.DeviceAuthorization, error) {
	return s.device(ctx, "device_code_hash = ?", hash)
}

// DeviceByUserCode is a device's sign-in waiting for the person, by the code
// they typed; ErrNotFound if none is waiting under it.
func (s *Store) DeviceByUserCode(ctx context.Context, hash string, now time.Time) (*model.DeviceAuthorization, error) {
	return s.device(ctx, "user_code_hash = ? AND status = ? AND expires_at > ?", hash, model.DevicePending, now)
}

func (s *Store) device(ctx context.Context, condition string, args ...any) (*model.DeviceAuthorization, error) {
	var device model.DeviceAuthorization
	if err := s.db.WithContext(ctx).Preload("Application").Where(condition, args...).First(&device).Error; err != nil {
		return nil, translate(err)
	}

	return &device, nil
}

// PolledDevice notes a device asked; asking too soon also slows it down.
func (s *Store) PolledDevice(ctx context.Context, device uuid.UUID, at time.Time, slowDown bool) error {
	changes := map[string]any{"last_polled_at": at}
	if slowDown {
		changes["interval"] = gorm.Expr("interval + ?", model.DeviceInterval)
	}

	return s.db.WithContext(ctx).Model(&model.DeviceAuthorization{}).Where("id = ?", device).Updates(changes).Error
}

// AnswerDevice approves or denies a device's sign-in, once, while it waits:
// false if it was already answered or has expired.
func (s *Store) AnswerDevice(ctx context.Context, device *model.DeviceAuthorization, approved bool, user, session uuid.UUID, authenticated, at time.Time) (bool, error) {
	changes := map[string]any{"status": model.DeviceDenied}
	if approved {
		changes = map[string]any{
			"status":           model.DeviceApproved,
			"user_id":          user,
			"session_id":       session,
			"authenticated_at": authenticated,
		}
	}

	result := s.db.WithContext(ctx).Model(&model.DeviceAuthorization{}).
		Where("id = ? AND status = ? AND expires_at > ?", device.ID, model.DevicePending, at).
		Updates(changes)

	return result.RowsAffected == 1, result.Error
}

// ClaimDevice spends an approved device sign-in: of two polls at the same
// moment only one gets the tokens.
func (s *Store) ClaimDevice(ctx context.Context, device uuid.UUID, at time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Model(&model.DeviceAuthorization{}).
		Where("id = ? AND status = ? AND used_at IS NULL", device, model.DeviceApproved).
		Update("used_at", at)

	return result.RowsAffected == 1, result.Error
}
