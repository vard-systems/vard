package api

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"loginer/internal/model"
)

// A pushed authorization request (RFC 9126) is sent server to server, and the
// browser carries only its request_uri: once, for the client that pushed it.
// An application can refuse every request that was not pushed.
func TestLivePushedAuthorizationRequests(t *testing.T) {
	f := newOAuthFixture(t)

	push := func(extra url.Values) tokenResult {
		form := url.Values{
			"response_type":         {"code"},
			"redirect_uri":          {shopRedirect},
			"scope":                 {"openid email"},
			"state":                 {"pushed-state"},
			"code_challenge":        {""},
			"code_challenge_method": {"S256"},
		}
		for key, values := range extra {
			form[key] = values
		}
		return f.token("/oauth2/par", f.clientID, f.clientSecret, form)
	}

	verifier, challenge := pkce(t)
	pushed := push(url.Values{"code_challenge": {challenge}})
	requestURI := pushed.str("request_uri")
	if pushed.status != http.StatusCreated || !strings.HasPrefix(requestURI, "urn:ietf:params:oauth:request_uri:") {
		t.Fatalf("pushing = %d %v", pushed.status, pushed.body)
	}

	// The browser carries the client_id and request_uri, nothing else.
	authorize := f.s.root + "/oauth2/authorize?" + url.Values{
		"client_id": {f.clientID}, "request_uri": {requestURI},
	}.Encode()

	b := f.s.browser()
	login := b.visit(authorize)
	if !strings.HasPrefix(login.String(), testAccountURL+"/login?") {
		t.Fatalf("the pushed request led to %s, want the sign-in page", login)
	}
	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	b.account(http.MethodPost, "/login", map[string]string{
		"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
	}, &out)
	back, _ := url.Parse(out.RedirectTo)
	if back.Query().Get("state") != "pushed-state" {
		t.Errorf("state came back as %q, want the pushed one", back.Query().Get("state"))
	}
	if tokens := f.exchange(back.Query().Get("code"), verifier); tokens.status != http.StatusOK {
		t.Fatalf("exchanging the code = %d %v", tokens.status, tokens.body)
	}

	if again := f.s.browser().visit(authorize); !strings.HasPrefix(again.String(), testAccountURL+"/error") {
		t.Errorf("a request_uri used twice led to %s, want the error page", again)
	}

	// Another client cannot spend this client's pushed request.
	_, challenge = pkce(t)
	theirs := push(url.Values{"code_challenge": {challenge}}).str("request_uri")
	stranger := f.s.root + "/oauth2/authorize?" + url.Values{"client_id": {"someone-else"}, "request_uri": {theirs}}.Encode()
	if at := f.s.browser().visit(stranger); !strings.HasPrefix(at.String(), testAccountURL+"/error") {
		t.Errorf("another client's request_uri led to %s, want the error page", at)
	}

	// What the authorization endpoint refuses, pushing refuses too — to the client.
	if bad := push(url.Values{"redirect_uri": {"https://elsewhere.example/cb"}}); bad.status != http.StatusBadRequest || bad.str("error") != "invalid_request" {
		t.Errorf("pushing an unregistered redirect_uri = %d %v", bad.status, bad.body)
	}
	if bad := f.token("/oauth2/par", f.clientID, "not-the-secret", url.Values{"response_type": {"code"}}); bad.status != http.StatusUnauthorized {
		t.Errorf("pushing with a wrong secret = %d, want 401", bad.status)
	}

	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "type": "web",
		"grant_types":                           []string{"authorization_code", "refresh_token"},
		"redirect_uris":                         []string{shopRedirect},
		"scopes":                                []string{"openid", "profile", "email", "offline_access", "roles"},
		"require_pushed_authorization_requests": true,
	}, nil)
	_, challenge = pkce(t)
	if at := f.s.browser().visit(f.authorizeURL(challenge, nil)); !strings.HasPrefix(at.String(), testAccountURL+"/error") {
		t.Errorf("a request not pushed, where pushing is required, led to %s", at)
	}
}

// A device that cannot show a sign-in page gets a short code for the person
// to enter elsewhere, and asks the token endpoint until they answer: told to
// wait, told to slow down when it asks too often, and given its tokens once
// they allow it — once.
func TestLiveDeviceAuthorization(t *testing.T) {
	f := newOAuthFixture(t)
	tvID, _, _ := f.register(map[string]any{
		"name": "Living room TV", "type": "native",
		"grant_types": []string{"urn:ietf:params:oauth:grant-type:device_code", "refresh_token"},
		"scopes":      []string{"openid", "email", "offline_access"},
	})

	start := func() (deviceCode, userCode string) {
		t.Helper()
		started := f.token("/oauth2/device_authorization", tvID, "", url.Values{"scope": {"openid email offline_access"}})
		if started.status != http.StatusOK || started.str("verification_uri") != testAccountURL+"/device" {
			t.Fatalf("starting a device = %d %v", started.status, started.body)
		}
		return started.str("device_code"), started.str("user_code")
	}
	poll := func(deviceCode string) tokenResult {
		return f.token("/oauth2/token", tvID, "", url.Values{
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {deviceCode},
		})
	}

	deviceCode, userCode := start()
	if got := poll(deviceCode).str("error"); got != "authorization_pending" {
		t.Errorf("asking before anyone answered = %s, want authorization_pending", got)
	}
	if got := poll(deviceCode).str("error"); got != "slow_down" {
		t.Errorf("asking again at once = %s, want slow_down", got)
	}

	// The person, signed in on their phone.
	b := f.s.browser()
	b.account(http.MethodPost, "/login", map[string]any{"email": adaEmail, "password": adaPassword}, nil)

	var refused struct {
		Code string `json:"code"`
	}
	if status := b.account(http.MethodGet, "/device?user_code=XXXX-XXXX", nil, &refused); status != http.StatusNotFound || refused.Code != "device_code_invalid" {
		t.Errorf("a code no device shows = %d %s", status, refused.Code)
	}

	// Typed as people type it: lower case, the dash left out.
	typed := strings.ToLower(strings.ReplaceAll(userCode, "-", ""))
	var shown struct {
		Device struct {
			Application struct {
				Name string `json:"name"`
			} `json:"application"`
			Scopes []struct {
				Name string `json:"name"`
			} `json:"scopes"`
		} `json:"device"`
	}
	if status := b.account(http.MethodGet, "/device?user_code="+typed, nil, &shown); status != http.StatusOK || shown.Device.Application.Name != "Living room TV" || len(shown.Device.Scopes) != 3 {
		t.Fatalf("the device page = %d %+v", status, shown.Device)
	}

	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": typed, "yes": true}, nil); status != http.StatusNoContent {
		t.Fatalf("allowing the device = %d", status)
	}
	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": typed, "yes": true}, nil); status != http.StatusNotFound {
		t.Errorf("answering the same code twice = %d, want 404", status)
	}

	tokens := poll(deviceCode)
	if tokens.status != http.StatusOK || tokens.str("id_token") == "" || tokens.str("refresh_token") == "" {
		t.Fatalf("asking once allowed = %d %v", tokens.status, tokens.body)
	}
	if _, claims := f.verifyJWT(tokens.str("id_token")); claims["sub"] != f.userID || claims["sid"] == nil {
		t.Errorf("the ID token is %v, want it for the person who allowed it, with their session", claims)
	}
	if again := poll(deviceCode); again.str("error") != "invalid_grant" {
		t.Errorf("asking after the tokens were given = %v, want invalid_grant", again.body)
	}

	deviceCode, userCode = start()
	b.account(http.MethodPost, "/device", map[string]any{"user_code": userCode, "yes": false}, nil)
	if got := poll(deviceCode).str("error"); got != "access_denied" {
		t.Errorf("a refused device = %s, want access_denied", got)
	}
}

// A client registers itself with its metadata (RFC 7591) — with an initial
// access token, an admin API token allowed applications.write, never without
// one — and becomes an application like any made in the panel.
func TestLiveDynamicClientRegistration(t *testing.T) {
	f := newOAuthFixture(t)

	cliID := f.adminCLI()
	adminToken := f.adminCLITokens(cliID)
	apiID, scopes := f.systemAPI(model.SystemAdminAPI)

	register := func(token string, metadata map[string]any) (int, map[string]any) {
		return f.bearer(http.MethodPost, f.s.root+"/oauth2/register", token, metadata)
	}
	web := map[string]any{
		"client_name":   "Partner portal",
		"redirect_uris": []string{"https://partner.example.com/callback"},
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"scope":         "openid email",
	}

	f.allow(cliID, apiID, scopes, model.PermApplicationsRead)
	if status, body := register(adminToken(), web); status != http.StatusForbidden || body["error"] != "insufficient_scope" {
		t.Errorf("registering without applications.write = %d %v", status, body)
	}
	if status, _ := register("not-a-token", web); status != http.StatusUnauthorized {
		t.Errorf("registering with a made-up token = %d, want 401", status)
	}

	f.allow(cliID, apiID, scopes, model.PermApplicationsWrite)
	token := adminToken()

	status, body := register(token, web)
	if status != http.StatusCreated || body["client_id"] == nil || body["client_secret"] == nil {
		t.Fatalf("registering a web client = %d %v", status, body)
	}
	if body["application_type"] != "web" || body["token_endpoint_auth_method"] != "client_secret_basic" {
		t.Errorf("the registered client is %v, want a confidential web client", body)
	}

	var listed struct {
		Applications []struct {
			Name string `json:"name"`
		} `json:"applications"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/applications?search=Partner", nil, &listed)
	if len(listed.Applications) != 1 {
		t.Errorf("the panel lists %d registered clients, want one", len(listed.Applications))
	}

	native := map[string]any{
		"client_name": "Desktop app", "application_type": "native",
		"redirect_uris": []string{"http://127.0.0.1:8123/callback"},
	}
	if status, body := register(token, native); status != http.StatusCreated || body["client_secret"] != nil || body["token_endpoint_auth_method"] != "none" {
		t.Errorf("registering a native client = %d %v, want a public client with no secret", status, body)
	}

	// What the panel refuses, registration refuses.
	bad := map[string]any{"redirect_uris": []string{"http://partner.example.com/callback"}}
	if status, body := register(token, bad); status != http.StatusBadRequest || body["error"] != "invalid_redirect_uri" {
		t.Errorf("registering a plain-http redirect = %d %v", status, body)
	}
	implicit := map[string]any{"redirect_uris": []string{"https://p.example.com/cb"}, "response_types": []string{"token"}}
	if status, body := register(token, implicit); status != http.StatusBadRequest || body["error"] != "invalid_client_metadata" {
		t.Errorf("registering for the implicit flow = %d %v", status, body)
	}
}
