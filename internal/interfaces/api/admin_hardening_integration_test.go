package api

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"
	"time"

	"vard/internal/brand"
)

// adminCookie is the administrator's session cookie this client holds.
func (c *client) adminCookie() string {
	c.s.t.Helper()

	root, err := url.Parse(c.s.adminRoot)
	if err != nil {
		c.s.t.Fatal(err)
	}
	for _, cookie := range c.http.Jar.Cookies(root) {
		if cookie.Name == brand.AdminSessionCookie {
			return cookie.Value
		}
	}
	return ""
}

// withAdminCookie is a fresh client holding only the given session cookie.
func (s *liveServer) withAdminCookie(value string) *client {
	s.t.Helper()

	jar, err := cookiejar.New(nil)
	if err != nil {
		s.t.Fatal(err)
	}
	root, _ := url.Parse(s.adminRoot)
	jar.SetCookies(root, []*http.Cookie{{Name: brand.AdminSessionCookie, Value: value, Path: "/"}})

	return &client{s: s, http: &http.Client{Jar: jar}}
}

// withAuthenticator gives the signed-in administrator an authenticator and
// returns its secret.
func (c *client) withAuthenticator(t *testing.T) string {
	t.Helper()

	var begun struct {
		Enrolment struct {
			Secret string `json:"secret"`
		} `json:"enrolment"`
	}
	c.must(http.StatusOK, http.MethodPost, "/mfa/totp", nil, &begun)
	c.must(http.StatusOK, http.MethodPost, "/mfa/totp/confirm", map[string]string{
		"code": codeFor(t, begun.Enrolment.Secret, time.Now().Add(-totp30)),
	}, nil)

	return begun.Enrolment.Secret
}

const totp30 = 30 * time.Second

// The cookie a password alone set is not the one that is signed in: passing
// the second factor issues a new session token and ends the old one, so a
// token planted or seen before the second factor is worth nothing after it.
func TestLiveTheAdminSessionTokenChangesWhenTheSecondFactorPasses(t *testing.T) {
	s := newLiveServer(t)
	root := s.superAdmin()
	secret := root.withAuthenticator(t)

	b := s.client()
	if _, answer := b.loginAnswer(superEmail, superPassword); answer.Next != "mfa" {
		t.Fatalf("login = %+v, want the second factor next", answer)
	}
	before := b.adminCookie()
	if before == "" {
		t.Fatal("the password set no session cookie")
	}

	b.must(http.StatusOK, http.MethodPost, "/auth/mfa", map[string]string{"code": codeFor(t, secret, time.Now())}, nil)
	after := b.adminCookie()

	if after == "" || after == before {
		t.Fatalf("the session cookie after the second factor = %q, want a new one", after)
	}
	if status := b.do(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Errorf("the new token = %d, want signed in", status)
	}
	if status := s.withAdminCookie(before).do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("the token from before the second factor = %d, want 401", status)
	}
}

// An administrator who walks away is signed out after the idle timeout, well
// before the session's twelve hours are up; one who keeps working is not.
func TestLiveAnIdleAdminSessionEnds(t *testing.T) {
	s := newLiveServer(t)
	root := s.superAdmin()

	var security struct {
		IdleTimeoutMinutes int `json:"idle_timeout_minutes"`
	}
	root.must(http.StatusOK, http.MethodGet, "/security", nil, &security)
	if security.IdleTimeoutMinutes != 30 {
		t.Errorf("the idle timeout = %d minutes, want 30 by default", security.IdleTimeoutMinutes)
	}

	working := s.client()
	if status := working.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("login = %d", status)
	}

	idle := func(minutes int) {
		t.Helper()
		if err := s.db.Exec("UPDATE admin_sessions SET last_seen_at = now() - make_interval(mins => ?)", minutes).Error; err != nil {
			t.Fatal(err)
		}
		// The sessions are kept in Redis too, when there is one; what was
		// written behind its back is read from the database again.
		if s.cache != nil {
			for _, db := range s.cache.Databases() {
				if err := db.Flush(context.Background()); err != nil {
					t.Fatal(err)
				}
			}
		}
	}

	idle(29)
	if status := working.do(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Fatalf("a session idle for 29 minutes = %d, want still signed in", status)
	}

	idle(31)
	if status := root.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("a session idle for 31 minutes = %d, want 401", status)
	}

	root2 := s.client()
	if status := root2.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("login = %d", status)
	}
	root2.must(http.StatusOK, http.MethodPatch, "/security", map[string]any{"idle_timeout_minutes": 120}, nil)
	idle(60)
	if status := root2.do(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Errorf("idle for an hour under a two-hour timeout = %d, want still signed in", status)
	}
}

// An administrator may sign in with a passkey as well as an authenticator.
// Adding one takes a code from the authenticator; signing in with it is one
// answer to one ceremony, and the session token changes as it does for a
// code.
func TestLiveAnAdministratorSignsInWithAPasskey(t *testing.T) {
	s := newLiveServer(t)
	root := s.superAdmin()

	var begun struct {
		Passkey map[string]any `json:"passkey"`
	}
	var refused problemBody
	if status := root.do(http.MethodPost, "/mfa/passkeys", map[string]string{"code": "123456"}, &refused); status != http.StatusBadRequest || refused.Code != "mfa_disabled" {
		t.Errorf("a passkey without an authenticator first = %d %s, want 400 mfa_disabled", status, refused.Code)
	}

	secret := root.withAuthenticator(t)
	if status := root.do(http.MethodPost, "/mfa/passkeys", map[string]string{"code": "000000"}, &refused); status != http.StatusBadRequest || refused.Code != "mfa_code_wrong" {
		t.Errorf("adding a passkey with a wrong code = %d %s, want 400 mfa_code_wrong", status, refused.Code)
	}
	root.must(http.StatusOK, http.MethodPost, "/mfa/passkeys", map[string]string{"code": codeFor(t, secret, time.Now())}, &begun)

	key := newVirtualKey(t)
	key.origin = s.cfg.AdminURL
	root.must(http.StatusOK, http.MethodPost, "/mfa/passkeys/finish", map[string]any{
		"label": "Laptop", "response": key.create(t, begun.Passkey),
	}, nil)

	var listed struct {
		MFA struct {
			Passkeys []struct {
				ID    string `json:"id"`
				Label string `json:"label"`
			} `json:"passkeys"`
		} `json:"mfa"`
	}
	root.must(http.StatusOK, http.MethodGet, "/mfa", nil, &listed)
	if len(listed.MFA.Passkeys) != 1 || listed.MFA.Passkeys[0].Label != "Laptop" {
		t.Errorf("the administrator's passkeys = %+v, want the laptop's", listed.MFA.Passkeys)
	}

	b := s.client()
	if _, answer := b.loginAnswer(superEmail, superPassword); answer.Next != "mfa" {
		t.Fatalf("login = %+v, want the second factor next", answer)
	}
	before := b.adminCookie()

	var offer struct {
		Passkey map[string]any `json:"passkey"`
	}
	b.must(http.StatusOK, http.MethodPost, "/auth/mfa/passkey", nil, &offer)
	answer := key.get(t, offer.Passkey)
	b.must(http.StatusOK, http.MethodPost, "/auth/mfa", map[string]any{"passkey": answer}, nil)

	if after := b.adminCookie(); after == "" || after == before {
		t.Errorf("the session cookie after a passkey = %q, want a new one", after)
	}
	if got := b.state(); got != "signed_in" {
		t.Errorf("after the passkey the session is %q, want signed_in", got)
	}

	// The same answer again, in another sign-in, is no use.
	again := s.client()
	again.loginAnswer(superEmail, superPassword)
	again.must(http.StatusOK, http.MethodPost, "/auth/mfa/passkey", nil, nil)
	if code := again.do(http.MethodPost, "/auth/mfa", map[string]any{"passkey": answer}, nil); code == http.StatusOK {
		t.Error("a passkey answer for another ceremony signed in")
	}

	// A key the account does not have is refused.
	stranger := newVirtualKey(t)
	stranger.origin = s.cfg.AdminURL
	stranger.user = key.user
	third := s.client()
	third.loginAnswer(superEmail, superPassword)
	third.must(http.StatusOK, http.MethodPost, "/auth/mfa/passkey", nil, &offer)
	if code := third.do(http.MethodPost, "/auth/mfa", map[string]any{"passkey": stranger.get(t, offer.Passkey)}, nil); code == http.StatusOK {
		t.Error("a passkey the administrator does not have signed in")
	}
}
