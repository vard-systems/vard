package auth

import (
	"context"
	"encoding/hex"

	"vard/internal/domain/model"
)

// Administrators' sign-ins are throttled like users', never locked: a lock
// is one a stranger who knows a username can hold shut. Passwords, codes and
// passkeys are all counted, before they are checked, against the one key:
// a keyed hash of the username typed, so a username nobody has is counted,
// and takes as long to answer, as one somebody has.

const throttlePurpose = "admin sign-in throttle"

func (s *Service) throttleKey(username string) string {
	tag := s.sealer.Tag(throttlePurpose, []byte(username))
	return "admin:" + hex.EncodeToString(tag[:16])
}

// reserve counts an attempt for `username` before it is checked; false when
// it has to wait.
func (s *Service) reserve(ctx context.Context, username string) (bool, error) {
	key := model.ThrottleKey{Key: s.throttleKey(username), Free: model.ThrottleFree}
	return s.store.ReserveSignIn(ctx, []model.ThrottleKey{key}, s.now())
}

// succeeded forgets the administrator's failed attempts.
func (s *Service) succeeded(ctx context.Context, admin *model.Admin) error {
	return s.store.ClearSignIn(ctx, s.throttleKey(admin.Username))
}

// Unthrottle forgets an administrator's failed attempts, for a super admin
// who has just set their password.
func (s *Service) Unthrottle(ctx context.Context, admin *model.Admin) error {
	return s.succeeded(ctx, admin)
}
