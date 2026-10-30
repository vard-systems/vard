package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"vard/internal/domain/model"
)

var ErrWrongPassword = errors.New("the current password is not right")

// ErrPasswordBreached is a new password known from a breach.
var ErrPasswordBreached = errors.New("the password is known from a breach")

// UseBreachCheck sets what says a new password is known from a breach; none
// checks nothing.
func (s *Service) UseBreachCheck(breached func(ctx context.Context, password string) bool) {
	s.breached = breached
}

// PasswordBreached says whether a password an administrator is choosing is
// known from a breach.
func (s *Service) PasswordBreached(ctx context.Context, password string) bool {
	return s.breached != nil && s.breached(ctx, password)
}

type Profile struct {
	FirstName string
	LastName  string
	Email     string
	AvatarURL string
}

func (s *Service) UpdateProfile(ctx context.Context, admin *model.Admin, profile Profile, currentPassword string) error {
	if err := s.withPassword(ctx, admin); err != nil {
		return err
	}

	if profile.Email != admin.Email {
		if err := s.checkPassword(ctx, admin, currentPassword); err != nil {
			return err
		}
	}

	admin.FirstName = profile.FirstName
	admin.LastName = profile.LastName
	admin.Email = profile.Email
	admin.Username = profile.Email
	admin.AvatarURL = profile.AvatarURL

	return s.store.SaveOwnAccount(ctx, admin)
}

func (s *Service) ChangePassword(ctx context.Context, admin *model.Admin, session uuid.UUID, current, next string) error {
	if err := s.withPassword(ctx, admin); err != nil {
		return err
	}

	if err := s.checkPassword(ctx, admin, current); err != nil {
		return err
	}
	if s.PasswordBreached(ctx, next) {
		return ErrPasswordBreached
	}

	if err := admin.SetPassword(next); err != nil {
		return err
	}

	return s.store.ChangeAdminPassword(ctx, admin, session, s.now())
}

func (s *Service) withPassword(ctx context.Context, admin *model.Admin) error {
	stored, err := s.store.AdminByID(ctx, admin.ID)
	if err != nil {
		return err
	}

	admin.PasswordHash = stored.PasswordHash

	return nil
}

// checkPassword says whether a password is the administrator's own. Each
// guess is counted before comparing, like a sign-in, so a session left open
// cannot be used to find the password; a waiting account is refused outright.
func (s *Service) checkPassword(ctx context.Context, admin *model.Admin, password string) error {
	allowed, err := s.reserve(ctx, admin)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrWrongPassword
	}

	if ok, _ := model.VerifyPassword(admin.PasswordHash, password); password == "" || !ok {
		return ErrWrongPassword
	}

	return s.succeeded(ctx, admin)
}
