package auth

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"loginer/internal/model"
)

var ErrWrongPassword = errors.New("the current password is not right")

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

	if err := admin.SetPassword(next); err != nil {
		return err
	}
	if err := s.store.SaveOwnAccount(ctx, admin); err != nil {
		return err
	}

	_, err := s.store.RevokeOtherAdminSessionsFor(ctx, admin.ID, session, s.now())
	return err
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
// cannot be used to find the password; a locked account is refused outright.
func (s *Service) checkPassword(ctx context.Context, admin *model.Admin, password string) error {
	allowed, err := s.store.ReserveAdminLogin(ctx, admin, s.now(), model.MaxFailedLogins, model.LockoutDuration)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrWrongPassword
	}

	if password == "" || bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return ErrWrongPassword
	}

	return s.store.ClearAdminFailedLogins(ctx, admin)
}
