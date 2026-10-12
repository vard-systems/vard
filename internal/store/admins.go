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

// ErrAdminExists keeps the setup endpoint closed once any administrator exists.
var ErrAdminExists = errors.New("an administrator already exists")

// firstAdminLock names the advisory lock CreateFirstAdmin holds. Any number
// will do, as long as nothing else locks the same one.
const firstAdminLock = brand.AdminLock

// AdminsExist reports whether anyone can sign in to the panel yet.
func (s *Store) AdminsExist(ctx context.Context) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Admin{}).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// CreateFirstAdmin writes the first administrator, refusing if one exists.
// Check and write share a locked transaction, so two simultaneous setups cannot
// both succeed.
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

// withAssignments preloads roles (panel-wide first) and confirmed factors.
func withAssignments(db *gorm.DB) *gorm.DB {
	return db.
		Preload("Assignments", func(db *gorm.DB) *gorm.DB {
			return db.Order("application_id NULLS FIRST, created_at")
		}).
		Preload("Assignments.Role").
		Preload("Assignments.Application").
		Preload("MFA", "confirmed_at IS NOT NULL")
}

// AdminByUsername loads an administrator and the roles they hold. It is what
// signing in starts with.
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

// AdminByID loads an administrator with their role assignments.
func (s *Store) AdminByID(ctx context.Context, id uuid.UUID) (*model.Admin, error) {
	var admin model.Admin
	err := withAssignments(s.db.WithContext(ctx)).
		First(&admin, "id = ?", id).Error
	if err != nil {
		return nil, translate(err)
	}

	return &admin, nil
}

// principal is how an administrator is cached: account, assignments and
// confirmed factors, without the password hash or factor secrets.
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

// MarkAdminSignedIn records when and from where an administrator last signed
// in, and forgets the wrong passwords that came before.
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

// ReserveAdminLogin counts an attempt before it is checked and reports false
// when the account is locked; the `max`th in a row locks it for `lockFor`.
// Counting first, in the database, holds the lockout against parallel attempts.
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

	// Sign-ins half way through read the account from the session database.
	s.forget(ctx, cache.Admins)

	return len(rows) > 0, nil
}

// ClearAdminFailedLogins gives back the attempts a right password took, for
// a sign-in that is not finished yet — one waiting for a code.
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

// AdminQuery is what a listing of administrators asks for: a search box, a
// filter, and a page.
type AdminQuery struct {
	// Search matches the username, the email or the name.
	Search string
	// Status filters on the account's status. Empty means every status.
	Status model.Status
	// Role keeps the administrators who hold this role, anywhere. Nil means
	// everyone.
	Role   *uuid.UUID
	Limit  int
	Offset int
}

// Admins returns a page of administrators, newest first, with their roles,
// along with how many match the query in total.
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

// CreateAdmin writes a new administrator with the roles they carry, in one
// transaction.
func (s *Store) CreateAdmin(ctx context.Context, admin *model.Admin) error {
	return translate(s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Create(admin).Error; err != nil {
			return err
		}

		return writeAssignments(tx, admin)
	}))
}

// SaveAdmin writes an administrator back and replaces the roles they hold with
// the ones they carry, in one transaction.
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

// writeAssignments inserts only the assignment rows.
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

// DeleteAdmin removes an administrator with their roles, sessions and factors.
// The activity log keeps their username.
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
