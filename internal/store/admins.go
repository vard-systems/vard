package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"loginer/internal/brand"
	"loginer/internal/cache"
	"loginer/internal/model"
)

var ErrAdminExists = errors.New("an administrator already exists")

const firstAdminLock = brand.AdminLock

func (s *Store) AdminsExist(ctx context.Context) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Admin{}).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (s *Store) CreateFirstAdmin(ctx context.Context, admin *model.Admin) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Without the lock, two concurrent setups would each count zero and
		// both insert.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", firstAdminLock).Error; err != nil {
			return err
		}

		var count int64
		if err := tx.Model(&model.Admin{}).Count(&count).Error; err != nil {
			return err
		}

		if count > 0 {
			return ErrAdminExists
		}

		var role model.AdminRole
		if err := tx.Where("name = ?", model.RoleSuperAdmin).First(&role).Error; err != nil {
			return err
		}

		admin.Assignments = nil
		if err := translate(tx.Omit(clause.Associations).Create(admin).Error); err != nil {
			return err
		}

		assignment := model.AdminRoleAssignment{AdminID: admin.ID, RoleID: role.ID, Role: role}
		if err := tx.Omit(clause.Associations).Create(&assignment).Error; err != nil {
			return err
		}

		admin.Assignments = []model.AdminRoleAssignment{assignment}

		return nil
	})
}

func withAssignments(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Assignments", func(db *gorm.DB) *gorm.DB {
			return db.Order("application_id NULLS FIRST, created_at")
		}).
		Preload("Assignments.Role").
		Preload("Assignments.Application").
		Preload("MFA", "confirmed_at IS NOT NULL")
}

func (s *Store) AdminByUsername(ctx context.Context, username string) (*model.Admin, error) {
	var admin model.Admin
	err := withAssignments(s.db.WithContext(ctx)).
		Where("username = ?", username).
		First(&admin).Error
	if err != nil {
		return nil, translate(err)
	}

	return &admin, nil
}

func (s *Store) AdminByID(ctx context.Context, id uuid.UUID) (*model.Admin, error) {
	var admin model.Admin
	err := withAssignments(s.db.WithContext(ctx)).
		First(&admin, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}

	return &admin, nil
}

type principal struct {
	Admin       model.Admin                 `json:"admin"`
	Assignments []model.AdminRoleAssignment `json:"assignments"`
	Factors     []model.MFA                 `json:"factors"`
}

// AdminPrincipal is AdminByID for authorisation, read through the session
// cache. It carries no password hash, so never save it or check a password
// against it.
func (s *Store) AdminPrincipal(ctx context.Context, id uuid.UUID) (*model.Admin, error) {
	kept, err := cached(ctx, s, cache.Admins, "id:"+id.String(), func() (principal, error) {
		admin, err := s.AdminByID(ctx, id)
		if err != nil {
			return principal{}, err
		}

		return principal{Admin: *admin, Assignments: admin.Assignments, Factors: admin.MFA}, nil
	})
	if err != nil {
		return nil, err
	}

	admin := kept.Admin
	admin.Assignments = kept.Assignments
	admin.MFA = kept.Factors

	return &admin, nil
}

func (s *Store) MarkAdminSignedIn(ctx context.Context, admin *model.Admin, at time.Time, ip string) error {
	err := s.db.WithContext(ctx).Model(admin).Updates(map[string]any{
		"last_login_at":      at,
		"last_login_ip":      ip,
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error
	if err == nil {
		s.forget(ctx, cache.Admins)
	}

	return err
}

func (s *Store) ReserveAdminLogin(ctx context.Context, admin *model.Admin, at time.Time, max int, lockFor time.Duration) (bool, error) {
	var rows []struct {
		LockedUntil *time.Time
	}

	err := s.db.WithContext(ctx).Raw(`
		UPDATE admins SET
			locked_until = CASE WHEN failed_login_count + 1 >= @max THEN @until ELSE locked_until END,
			failed_login_count = CASE WHEN failed_login_count + 1 >= @max THEN 0 ELSE failed_login_count + 1 END
		WHERE id = @id AND (locked_until IS NULL OR locked_until <= @now)
		RETURNING locked_until`,
		map[string]any{"max": max, "until": at.Add(lockFor), "id": admin.ID, "now": at},
	).Scan(&rows).Error
	if err != nil {
		return false, err
	}

	s.forget(ctx, cache.Admins)

	return len(rows) > 0, nil
}

func (s *Store) ClearAdminFailedLogins(ctx context.Context, admin *model.Admin) error {
	err := s.db.WithContext(ctx).Model(admin).Updates(map[string]any{
		"failed_login_count": 0,
		"locked_until":       nil,
	}).Error
	if err == nil {
		s.forget(ctx, cache.Admins)
	}

	return err
}

type AdminQuery struct {
	Search string
	Status model.Status
	Role   *uuid.UUID
	Limit  int
	Offset int
}

func (s *Store) Admins(ctx context.Context, q AdminQuery) ([]model.Admin, int64, error) {
	query := s.db.WithContext(ctx).Model(&model.Admin{})

	if search := strings.TrimSpace(q.Search); search != "" {
		like := contains(search)
		query = query.Where(
			`LOWER(username) LIKE ? OR LOWER(email) LIKE ? OR LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ?`,
			like, like, like, like,
		)
	}

	if q.Status != "" {
		query = query.Where("status = ?", q.Status)
	}

	if q.Role != nil {
		query = query.Where("id IN (SELECT admin_id FROM admin_role_assignments WHERE role_id = ?)", *q.Role)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var admins []model.Admin
	err := withAssignments(query).
		Order("created_at DESC").
		Limit(q.Limit).
		Offset(q.Offset).
		Find(&admins).Error

	return admins, total, err
}

func (s *Store) CreateAdmin(ctx context.Context, admin *model.Admin) error {
	return translate(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(admin).Error; err != nil {
			return err
		}

		return writeAssignments(tx, admin)
	}))
}

func (s *Store) SaveAdmin(ctx context.Context, admin *model.Admin) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(admin).Error; err != nil {
			return err
		}

		if err := tx.Where("admin_id = ?", admin.ID).Delete(&model.AdminRoleAssignment{}).Error; err != nil {
			return err
		}

		return writeAssignments(tx, admin)
	})), cache.Admins)
}

// SaveOwnAccount writes only name, address and password, so a profile change
// can never touch roles. A taken address is ErrDuplicate.
func (s *Store) SaveOwnAccount(ctx context.Context, admin *model.Admin) error {
	return s.forgetting(ctx, translate(s.db.WithContext(ctx).
		Model(admin).
		Select("first_name", "last_name", "email", "username", "avatar_url", "password_hash").
		Updates(admin).Error), cache.Admins)
}

func writeAssignments(tx *gorm.DB, admin *model.Admin) error {
	for i := range admin.Assignments {
		admin.Assignments[i].ID = uuid.Nil
		admin.Assignments[i].AdminID = admin.ID

		if err := tx.Omit(clause.Associations).Create(&admin.Assignments[i]).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) DeleteAdmin(ctx context.Context, admin *model.Admin) error {
	return s.forgetting(ctx, s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		statements := []string{
			"DELETE FROM admin_role_assignments WHERE admin_id = @id",
			"DELETE FROM admin_sessions WHERE admin_id = @id",
			"DELETE FROM mfa_factors WHERE admin_id = @id",
		}

		for _, statement := range statements {
			if err := tx.Exec(statement, map[string]any{"id": admin.ID}).Error; err != nil {
				return err
			}
		}

		return tx.Unscoped().Delete(admin).Error
	}), cache.Admins)
}
