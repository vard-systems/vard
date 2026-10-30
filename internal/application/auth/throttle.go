package auth

import (
	"context"

	"vard/internal/domain/model"
)

// Administrators' sign-ins are throttled like users', never locked: a lock
// is one a stranger who knows a username can hold shut. Passwords, codes and
// passkeys are all counted, before they are checked, against the one key.

func adminThrottleKey(admin *model.Admin) string {
	return "admin:" + admin.ID.String()
}

// reserve counts an attempt before it is checked; false when it has to wait.
func (s *Service) reserve(ctx context.Context, admin *model.Admin) (bool, error) {
	key := model.ThrottleKey{Key: adminThrottleKey(admin), Free: model.ThrottleFree}
	return s.store.ReserveSignIn(ctx, []model.ThrottleKey{key}, s.now())
}

// succeeded forgets the administrator's failed attempts.
func (s *Service) succeeded(ctx context.Context, admin *model.Admin) error {
	return s.store.ClearSignIn(ctx, adminThrottleKey(admin))
}

// Unthrottle forgets an administrator's failed attempts, for a super admin
// who has just set their password.
func (s *Service) Unthrottle(ctx context.Context, admin *model.Admin) error {
	return s.succeeded(ctx, admin)
}
