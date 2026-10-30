package api

import (
	"net/http"
	"strings"
	"testing"
)

// raw sends one request with exactly these headers, and returns the status.
func raw(t *testing.T, method, url, body string, headers map[string]string) int {
	t.Helper()

	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	return res.StatusCode
}

// The admin API answers only on the admin listener; the provider only on the
// public one.
func TestLiveAdminAPIIsNotOnThePublicServer(t *testing.T) {
	s := newLiveServer(t)
	s.superAdmin()

	if status := raw(t, http.MethodGet, s.root+"/api/v1/admin/setup", "", nil); status != http.StatusNotFound {
		t.Errorf("public GET /api/v1/admin/setup = %d, want 404", status)
	}
	login := `{"username":"root@example.com","password":"root-password-1"}`
	if status := raw(t, http.MethodPost, s.root+"/api/v1/admin/auth/login", login, map[string]string{"Content-Type": "application/json"}); status != http.StatusNotFound {
		t.Errorf("public admin login = %d, want 404", status)
	}
	if status := raw(t, http.MethodPost, s.adminRoot+"/api/v1/admin/auth/login", login, map[string]string{"Content-Type": "application/json"}); status != http.StatusOK {
		t.Errorf("admin login on the admin server = %d, want 200", status)
	}
	if status := raw(t, http.MethodGet, s.adminRoot+"/.well-known/openid-configuration", "", nil); status != http.StatusNotFound {
		t.Errorf("admin server discovery = %d, want 404", status)
	}
}

// A page on another origin — a sibling subdomain included — cannot change
// anything with a session cookie, by script or by form.
func TestLiveCookieAPIsRefuseOtherOrigins(t *testing.T) {
	s := newLiveServer(t)
	s.superAdmin()

	login := `{"email":"ada@example.com","password":"whatever-password"}`

	cases := []struct {
		name    string
		url     string
		body    string
		headers map[string]string
		want    int
	}{
		{"sibling subdomain, account API", s.root + "/api/v1/account/login", login,
			map[string]string{"Content-Type": "application/json", "Origin": "https://blog.mywebsite.com"}, http.StatusForbidden},
		{"form posting text/plain, account API", s.root + "/api/v1/account/login", login,
			map[string]string{"Content-Type": "text/plain", "Origin": testAccountURL}, http.StatusUnsupportedMediaType},
		{"the id app itself", s.root + "/api/v1/account/login", login,
			map[string]string{"Content-Type": "application/json", "Origin": testAccountURL}, http.StatusUnauthorized},
		{"the id app's origin on the admin API", s.adminRoot + "/api/v1/admin/auth/login", `{"username":"root@example.com","password":"root-password-1"}`,
			map[string]string{"Content-Type": "application/json", "Origin": testAccountURL}, http.StatusForbidden},
		{"the console itself", s.adminRoot + "/api/v1/admin/auth/login", `{"username":"root@example.com","password":"root-password-1"}`,
			map[string]string{"Content-Type": "application/json", "Origin": "http://admin.test"}, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if status := raw(t, http.MethodPost, tc.url, tc.body, tc.headers); status != tc.want {
				t.Errorf("status = %d, want %d", status, tc.want)
			}
		})
	}
}

// The provider's own endpoints have a limit of their own, far looser than the
// sign-in pages'. An application's backend may be exchanging codes for a whole
// company from one address, so the two budgets are separate and this one is
// generous; what it stops is a caller asking for ever, not a busy client.
func TestLiveTokenEndpointHasItsOwnRateLimit(t *testing.T) {
	s := newLiveServerLimited(t, 1)

	form := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	body := "grant_type=client_credentials&client_id=nobody&client_secret=nothing"

	for i := range tokenLimitMultiple {
		if status := raw(t, http.MethodPost, s.root+"/oauth2/token", body, form); status != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401 inside the limit", i+1, status)
		}
	}

	if status := raw(t, http.MethodPost, s.root+"/oauth2/token", body, form); status != http.StatusTooManyRequests {
		t.Errorf("attempt %d = %d, want 429", tokenLimitMultiple+1, status)
	}

	// Spending that budget did not spend the sign-in pages': they count apart.
	signIn := `{"email":"nobody@example.com","password":"guess-password"}`
	if status := raw(t, http.MethodPost, s.root+"/api/v1/account/login", signIn, map[string]string{"Content-Type": "application/json"}); status != http.StatusUnauthorized {
		t.Errorf("signing in after the token endpoint's limit = %d, want 401", status)
	}
}

// One address guessing across accounts is slowed down, whatever the account.
func TestLiveSignInIsRateLimited(t *testing.T) {
	s := newLiveServerLimited(t, 3)

	body := func(email string) string { return `{"email":"` + email + `","password":"guess-password"}` }
	json := map[string]string{"Content-Type": "application/json"}

	for i, email := range []string{"a@example.com", "b@example.com", "c@example.com"} {
		if status := raw(t, http.MethodPost, s.root+"/api/v1/account/login", body(email), json); status != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401 inside the limit", i+1, status)
		}
	}

	if status := raw(t, http.MethodPost, s.root+"/api/v1/account/login", body("d@example.com"), json); status != http.StatusTooManyRequests {
		t.Errorf("fourth attempt = %d, want 429", status)
	}

	// Reading is not limited.
	if status := raw(t, http.MethodGet, s.root+"/.well-known/openid-configuration", "", nil); status != http.StatusOK {
		t.Errorf("discovery after the limit = %d, want 200", status)
	}
}
