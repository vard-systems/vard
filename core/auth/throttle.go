package auth

import (
	"context"
	"encoding/hex"

	"vard/core/model"
)

// Administrators' sign-ins are throttled like users', never locked: a lock
// is one a stranger who knows a username can hold shut. Passwords, codes and
// passkeys are all counted, before they are checked, against the one key:
// a keyed hash of the username typed, so a username nobody has is counted,
// and takes as long to answer, as one somebody has. A browser that has
// signed in as the administrator before (model.KnownBrowser) is counted on a
// budget of its own instead, so a stranger failing at the username keeps only
// strangers waiting.

const (
	throttlePurpose     = "admin sign-in throttle"
	knownBrowserPurpose = "admin known browser"
)

// usernameKey is the budget of every browser not known to the username.
func (s *Service) usernameKey(username string) string {
	tag := s.sealer.Tag(throttlePurpose, []byte(username))
	return "admin:" + hex.EncodeToString(tag[:16])
}

// throttleKey is what an attempt at `username` from this request is counted
// against: the browser's own budget when it is known to the username, the
// username's otherwise.
func (s *Service) throttleKey(username string, req Request) string {
	browser := model.KnownBrowser(s.sealer, knownBrowserPurpose, req.KnownBrowser, username, s.now())
	if browser == "" {
		return s.usernameKey(username)
	}

	tag := s.sealer.Tag(throttlePurpose, []byte("browser\x00"+browser+"\x00"+username))
	return "admin-browser:" + hex.EncodeToString(tag[:16])
}

// reserve counts an attempt for `username` before it is checked; false when
// it has to wait.
func (s *Service) reserve(ctx context.Context, username string, req Request) (bool, error) {
	key := model.ThrottleKey{Key: s.throttleKey(username, req), Free: model.ThrottleFree}
	return s.store.ReserveSignIn(ctx, []model.ThrottleKey{key}, s.now())
}

// succeeded forgets the failed attempts this request was counted against.
func (s *Service) succeeded(ctx context.Context, admin *model.Admin, req Request) error {
	return s.store.ClearSignIn(ctx, s.throttleKey(admin.Username, req))
}

// Unthrottle forgets the failed attempts at an administrator's username, for
// a super admin who has just set their password.
func (s *Service) Unthrottle(ctx context.Context, admin *model.Admin) error {
	return s.store.ClearSignIn(ctx, s.usernameKey(admin.Username))
}

// KnownBrowser is the cookie value that marks this browser as one that has
// signed in as the administrator, for a sign-in that has just finished.
func (s *Service) KnownBrowser(admin *model.Admin) (string, error) {
	return model.NewKnownBrowser(s.sealer, knownBrowserPurpose, admin.Username, s.now())
}
