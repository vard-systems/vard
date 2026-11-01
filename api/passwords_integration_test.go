package api

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"vard/core/config"
)

// A password hashed with bcrypt before the move to Argon2id still signs in,
// and is hashed again with Argon2id as it does; a wrong one changes nothing.
func TestLiveABcryptPasswordIsRehashedWhenItSignsIn(t *testing.T) {
	f := newOAuthFixture(t)

	old, err := bcrypt.GenerateFromPassword([]byte(adaPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", string(old), f.userID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.Exec("UPDATE admins SET password_hash = ? WHERE email = ?", mustBcrypt(t, superPassword), superEmail).Error; err != nil {
		t.Fatal(err)
	}

	hash := func(table, where string, arg any) string {
		t.Helper()
		var stored string
		if err := f.s.db.Table(table).Select("password_hash").Where(where, arg).Scan(&stored).Error; err != nil {
			t.Fatal(err)
		}
		return stored
	}

	f.s.browser().account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": "a-wrong-password"}, nil)
	if got := hash("users", "id = ?", f.userID); got != string(old) {
		t.Error("a wrong password changed the stored hash")
	}

	if status := f.s.browser().account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": adaPassword}, nil); status != http.StatusOK {
		t.Fatalf("signing in with the bcrypt password = %d", status)
	}
	if got := hash("users", "id = ?", f.userID); !strings.HasPrefix(got, "$argon2id$") {
		t.Errorf("the user's hash after signing in = %.20s…, want Argon2id", got)
	}
	if status := f.s.browser().account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": adaPassword}, nil); status != http.StatusOK {
		t.Errorf("signing in with the rehashed password = %d", status)
	}

	if status := f.s.client().login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("the administrator signing in with a bcrypt password = %d", status)
	}
	if got := hash("admins", "email = ?", superEmail); !strings.HasPrefix(got, "$argon2id$") {
		t.Errorf("the administrator's hash after signing in = %.20s…, want Argon2id", got)
	}
}

func mustBcrypt(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

// pwnedAPI answers the range API with the given passwords among its suffixes,
// and counts what it was asked.
func pwnedAPI(t *testing.T, breached ...string) (*httptest.Server, *[]string) {
	t.Helper()

	asked := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		prefix := strings.TrimPrefix(r.URL.Path, "/range/")
		asked = append(asked, prefix)
		fmt.Fprintln(w, "0018A45C4D1DEF81644B54AB7F969B88D65:1")
		for _, password := range breached {
			sum := sha1.Sum([]byte(password))
			digest := strings.ToUpper(hex.EncodeToString(sum[:]))
			if digest[:5] == prefix {
				fmt.Fprintf(w, "%s:42\n", digest[5:])
			}
		}
	}))
	t.Cleanup(server.Close)

	return server, &asked
}

// A password known from a breach is refused where a person chooses one, and
// only five characters of its hash ever leave the server. An API that does
// not answer lets the password through.
func TestLiveABreachedPasswordIsRefused(t *testing.T) {
	const breached = "correct-horse-battery"
	api, asked := pwnedAPI(t, breached)

	f := newOAuthFixtureWith(t, func(cfg *config.Config) { cfg.PwnedPasswordsURL = api.URL })
	allowUnverifiedAddresses(t, f.super)
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "type": "web",
		"grant_types":        []string{"authorization_code", "refresh_token"},
		"redirect_uris":      []string{shopRedirect},
		"scopes":             []string{"openid", "profile", "email", "offline_access", "roles"},
		"allow_registration": true,
	}, nil)

	register := func(password string) (int, string) {
		b := f.s.browser()
		_, challenge := pkce(t)
		login := b.visit(f.authorizeURL(challenge, nil))
		var out problemBody
		status := b.account(http.MethodPost, "/register", map[string]any{
			"request": login.Query().Get("request"), "email": randomHex(t, 4) + "@example.com", "password": password,
		}, &out)
		return status, out.Code
	}

	if status, code := register(breached); status != http.StatusBadRequest || code != "password_breached" {
		t.Errorf("registering with a breached password = %d %s, want 400 password_breached", status, code)
	}
	for _, prefix := range *asked {
		if len(prefix) != 5 {
			t.Errorf("the API was sent %q, want only five characters of the hash", prefix)
		}
	}
	if status, code := register("a-password-nobody-has-1"); status != http.StatusOK {
		t.Errorf("registering with a password the API does not list = %d %s, want 200", status, code)
	}

	var refused problemBody
	if status := f.super.do(http.MethodPost, "/me/password", map[string]string{"current_password": superPassword, "new_password": breached}, &refused); status != http.StatusBadRequest || refused.Code != "admin_password_breached" {
		t.Errorf("an administrator choosing a breached password = %d %s, want 400 admin_password_breached", status, refused.Code)
	}
	if status := f.super.do(http.MethodPost, "/users", map[string]any{
		"email": "given@example.com", "password": breached, "confirm_password": breached,
	}, &refused); status != http.StatusBadRequest || refused.Code != "user_password_breached" {
		t.Errorf("an administrator giving a user a breached password = %d %s, want 400 user_password_breached", status, refused.Code)
	}

	api.Close()
	if status, code := register(breached); status != http.StatusOK {
		t.Errorf("a breached password while the API is down = %d %s, want it let through", status, code)
	}
}

// A password shorter than the floor is refused before anything is made.
func TestLiveAShortPasswordIsRefused(t *testing.T) {
	f := newOAuthFixture(t)
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "type": "web",
		"grant_types":        []string{"authorization_code", "refresh_token"},
		"redirect_uris":      []string{shopRedirect},
		"scopes":             []string{"openid", "profile", "email", "offline_access", "roles"},
		"allow_registration": true,
	}, nil)

	b := f.s.browser()
	_, challenge := pkce(t)
	login := b.visit(f.authorizeURL(challenge, nil))
	var out problemBody
	if status := b.account(http.MethodPost, "/register", map[string]any{
		"request": login.Query().Get("request"), "email": "short@example.com", "password": "eleven-char",
	}, &out); status != http.StatusBadRequest || out.Code != "password_too_short" {
		t.Errorf("an eleven-character password = %d %s, want 400 password_too_short", status, out.Code)
	}
}
