package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"testing"

	"vard/core/brand"
	"vard/core/model"
)

// These tests call this server's own two APIs the way other software does:
// the admin API with admin-cli's client credentials token, and the account
// API with the token an application got for a signed-in user.

// systemAPI finds one of the two system APIs the way the panel does, with its
// scope ids by name.
func (f *oauthFixture) systemAPI(system string) (id string, scopes map[string]string) {
	var out struct {
		APIs []struct {
			ID     string `json:"id"`
			System string `json:"system"`
			Scopes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"scopes"`
		} `json:"apis"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/apis", nil, &out)

	for _, api := range out.APIs {
		if api.System != system {
			continue
		}
		scopes = map[string]string{}
		for _, scope := range api.Scopes {
			scopes[scope.Name] = scope.ID
		}
		return api.ID, scopes
	}

	f.s.t.Fatalf("there is no %s system API", system)
	return "", nil
}

// allow sets the scopes of a system API an application may ask for.
func (f *oauthFixture) allow(appID, apiID string, scopes map[string]string, names ...string) {
	ids := []string{}
	for _, name := range names {
		ids = append(ids, scopes[name])
	}
	f.super.must(http.StatusOK, http.MethodPut, "/applications/"+appID+"/apis/"+apiID, map[string]any{"scopes": ids}, nil)
}

// bearer calls a URL with an access token and no cookie.
func (f *oauthFixture) bearer(method, target, token string, body any) (int, map[string]any) {
	f.s.t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			f.s.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, target, reader)
	if err != nil {
		f.s.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		f.s.t.Fatal(err)
	}
	defer res.Body.Close()

	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// adminCLITokens gives admin-cli the secret an administrator has to give it
// before it can do anything, and returns how to ask for a client credentials
// token for the admin API with it.
func (f *oauthFixture) adminCLITokens(cliID string) func() string {
	t := f.s.t
	t.Helper()

	var rotated struct {
		ClientSecret string `json:"client_secret"`
	}
	f.super.must(http.StatusOK, http.MethodPost, "/applications/"+cliID+"/secret", nil, &rotated)

	return func() string {
		t.Helper()
		tokens := f.token("/oauth2/token", model.AdminCLIClientID, rotated.ClientSecret, url.Values{
			"grant_type": {"client_credentials"}, "audience": {brand.AdminAPIIdentifier},
		})
		if tokens.status != http.StatusOK {
			t.Fatalf("admin-cli token = %d %v", tokens.status, tokens.body)
		}
		return tokens.str("access_token")
	}
}

func TestLiveAdminCLIManagesUsers(t *testing.T) {
	f := newOAuthFixture(t)
	admin := f.s.url

	cliID := f.adminCLI()
	adminToken := f.adminCLITokens(cliID)
	apiID, scopes := f.systemAPI(model.SystemAdminAPI)
	f.allow(cliID, apiID, scopes, model.PermUsersRead)

	// Asking for no scope gets every scope it is allowed: they are defaults.
	token := adminToken()
	if _, claims := f.verifyJWT(token); claims["scope"] != model.PermUsersRead {
		t.Errorf("admin-cli token scope = %v, want %s", claims["scope"], model.PermUsersRead)
	}

	tests := []struct {
		name         string
		method, path string
		body         any
		want         int
		code         string
	}{
		{name: "reading users is what users.read allows", method: http.MethodGet, path: "/users", want: http.StatusOK},
		{name: "creating one takes users.write", method: http.MethodPost, path: "/users",
			body: map[string]any{"email": "bob@example.com"}, want: http.StatusForbidden, code: "forbidden"},
		{name: "a super admin's route is a person's", method: http.MethodGet, path: "/admins",
			want: http.StatusUnauthorized, code: "not_signed_in"},
		{name: "an administrator's own account is a person's", method: http.MethodGet, path: "/me",
			want: http.StatusUnauthorized, code: "not_signed_in"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, body := f.bearer(tt.method, admin+tt.path, token, tt.body)
			if status != tt.want || (tt.code != "" && body["code"] != tt.code) {
				t.Errorf("%s %s = %d %v, want %d %s", tt.method, tt.path, status, body, tt.want, tt.code)
			}
		})
	}

	// Given users.write, it creates a user, and the activity log names it.
	f.allow(cliID, apiID, scopes, model.PermUsersRead, model.PermUsersWrite)
	if status, body := f.bearer(http.MethodPost, admin+"/users", adminToken(), map[string]any{
		"email": "bob@example.com", "password": "bob-password-1", "confirm_password": "bob-password-1",
	}); status != http.StatusCreated {
		t.Fatalf("create a user with admin-cli = %d %v", status, body)
	}

	var logs struct {
		Logs []struct {
			ActorEmail string `json:"actor"`
			Action     string `json:"action"`
		} `json:"logs"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/logs?action=user.created", nil, &logs)
	if len(logs.Logs) == 0 || logs.Logs[0].ActorEmail != model.AdminCLIClientID {
		t.Errorf("the log of the user admin-cli made = %+v, want admin-cli as the actor", logs.Logs)
	}

	// Taking a scope away takes effect on the next call, not at expiry.
	token = adminToken()
	f.allow(cliID, apiID, scopes)
	if status, _ := f.bearer(http.MethodGet, admin+"/users", token, nil); status != http.StatusForbidden {
		t.Errorf("reading users after users.read was taken away = %d, want 403", status)
	}

	// Turned off, admin-cli's tokens stop working at once.
	f.allow(cliID, apiID, scopes, model.PermUsersRead)
	token = adminToken()
	var current struct {
		Application map[string]any `json:"application"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/applications/"+cliID, nil, &current)
	current.Application["is_enabled"] = false
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+cliID, current.Application, nil)
	if status, body := f.bearer(http.MethodGet, admin+"/users", token, nil); status != http.StatusUnauthorized || body["code"] != "token_refused" {
		t.Errorf("a disabled admin-cli's token = %d %v, want 401 token_refused", status, body)
	}

	// A person's token for the admin API is refused, whatever it carries: the
	// admin API acts for administrators, and signing in to an application
	// does not make anyone one.
	f.allow(f.appID, apiID, scopes, model.PermUsersRead)
	b := f.s.browser()
	verifier, challenge := pkce(t)
	back := f.signInWith(b, challenge, url.Values{"audience": {brand.AdminAPIIdentifier}, "scope": {"openid " + model.PermUsersRead}})
	person := f.exchange(back.Query().Get("code"), verifier)
	if status, body := f.bearer(http.MethodGet, admin+"/users", person.str("access_token"), nil); status != http.StatusUnauthorized || body["code"] != "token_refused" {
		t.Errorf("a user's admin API token = %d %v, want 401 token_refused", status, body)
	}

	// Both are the server's own, and stay.
	if status := f.super.do(http.MethodDelete, "/applications/"+cliID, nil, nil); status != http.StatusConflict {
		t.Errorf("delete admin-cli = %d, want 409", status)
	}
	if status := f.super.do(http.MethodDelete, "/apis/"+apiID, nil, nil); status != http.StatusConflict {
		t.Errorf("delete the admin API = %d, want 409", status)
	}
}

// The organisation's settings are managed like any other setting a role may
// hold: a token with organization.write changes them, one without it does not,
// and the activity log names the token's application as the one who did.
func TestLiveAdminCLIChangesTheOrganization(t *testing.T) {
	f := newOAuthFixture(t)
	admin := f.s.url

	cliID := f.adminCLI()
	adminToken := f.adminCLITokens(cliID)
	apiID, scopes := f.systemAPI(model.SystemAdminAPI)
	change := map[string]any{"support_email": "help@acme.example.com"}

	f.allow(cliID, apiID, scopes, model.PermOrganizationRead)
	if status, body := f.bearer(http.MethodPatch, admin+"/organization", adminToken(), change); status != http.StatusForbidden || body["code"] != "forbidden" {
		t.Errorf("change the organization with organization.read = %d %v, want 403 forbidden", status, body)
	}

	f.allow(cliID, apiID, scopes, model.PermOrganizationRead, model.PermOrganizationWrite)
	status, body := f.bearer(http.MethodPatch, admin+"/organization", adminToken(), change)
	if status != http.StatusOK {
		t.Fatalf("change the organization with organization.write = %d %v", status, body)
	}
	if organization, _ := body["organization"].(map[string]any); organization["support_email"] != "help@acme.example.com" {
		t.Errorf("after the change = %v, want the new support address", body)
	}

	var logs struct {
		Logs []struct {
			Actor string `json:"actor"`
		} `json:"logs"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/logs?action=organization.updated", nil, &logs)
	if len(logs.Logs) == 0 || logs.Logs[0].Actor != model.AdminCLIClientID {
		t.Errorf("the log of the change = %+v, want admin-cli as the actor", logs.Logs)
	}
}

func TestLiveAccountAPIWithAUserToken(t *testing.T) {
	f := newOAuthFixture(t)
	account := f.s.root + "/api/v1/account"

	apiID, scopes := f.systemAPI(model.SystemAccountAPI)
	f.allow(f.appID, apiID, scopes, model.ScopeAccountRead, model.ScopeAccountWrite)

	signIn := func(scope string) string {
		t.Helper()
		b := f.s.browser()
		verifier, challenge := pkce(t)
		back := f.signInWith(b, challenge, url.Values{"audience": {brand.AccountAPIIdentifier}, "scope": {scope}})
		tokens := f.exchange(back.Query().Get("code"), verifier)
		if tokens.status != http.StatusOK {
			t.Fatalf("sign in for the account API = %d %v", tokens.status, tokens.body)
		}
		return tokens.str("access_token")
	}

	reader := signIn("openid")
	status, me := f.bearer(http.MethodGet, account+"/me", reader, nil)
	user, _ := me["user"].(map[string]any)
	if status != http.StatusOK || user["email"] != adaEmail {
		t.Fatalf("GET /me with an account.read token = %d %v", status, me)
	}

	if status, body := f.bearer(http.MethodPatch, account+"/me", reader, map[string]any{"first_name": "Augusta"}); status != http.StatusForbidden || body["code"] != "account_scope_missing" {
		t.Errorf("PATCH /me with account.read alone = %d %v, want 403 account_scope_missing", status, body)
	}

	writer := signIn("openid " + model.ScopeAccountWrite)
	if status, body := f.bearer(http.MethodPatch, account+"/me", writer, map[string]any{"first_name": "Augusta", "last_name": "Lovelace"}); status != http.StatusOK {
		t.Errorf("PATCH /me with account.write = %d %v", status, body)
	}

	// Even with account.write and the password, a token may not move the
	// account to another address: that and a password reset are the account.
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, f.super), map[string]any{"allow_email_change": true}, nil)
	if status, body := f.bearer(http.MethodPost, account+"/email", writer, map[string]any{
		"email": "someone.else@example.com", "current_password": adaPassword,
	}); status != http.StatusForbidden || body["code"] != "email_change_not_offered" {
		t.Errorf("POST /email with an account.write token = %d %v, want 403 email_change_not_offered", status, body)
	}

	// Nor may a token accept terms, allow an application or start a step-up
	// for its user: those pages answer only to the person's own browser.
	for _, path := range []string{"/consent", "/agreements", "/device", "/step-up"} {
		if status, body := f.bearer(http.MethodPost, account+path, writer, map[string]any{
			"request": "any", "user_code": "BCDF-GHJK", "yes": true,
		}); status != http.StatusForbidden || body["code"] != "browser_session_required" {
			t.Errorf("POST %s with an account.write token = %d %v, want 403 browser_session_required", path, status, body)
		}
	}

	// A token for another API is not an account API token.
	b := f.s.browser()
	verifier, challenge := pkce(t)
	orders := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	if status, body := f.bearer(http.MethodGet, account+"/me", orders.str("access_token"), nil); status != http.StatusUnauthorized || body["code"] != "account_token_refused" {
		t.Errorf("GET /me with an orders API token = %d %v, want 401 account_token_refused", status, body)
	}
}
