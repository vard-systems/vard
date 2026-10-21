package api

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"maps"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strings"
	"testing"
)

// These tests walk the provider the way a real application and a real browser
// do: the application sends the browser to authorize, the browser signs in
// through the account endpoints the sign-in page calls, and the application
// exchanges the code, refreshes, calls userinfo and signs out.

const (
	shopRedirect = "https://shop.example.com/callback"
	shopLoggedIn = "https://shop.example.com/"
	ordersAPI    = "https://api.example.com/orders"
	adaEmail     = "ada@example.com"
	adaPassword  = "ada-password-1"
)

// oauthFixture is an installation with a web application, an API it may call,
// and a user holding a role that grants one of the API's scopes.
type oauthFixture struct {
	s            *liveServer
	super        *client
	clientID     string
	clientSecret string
	appID        string
	apiID        string
	scopeIDs     map[string]string
	userID       string
}

func newOAuthFixture(t *testing.T) *oauthFixture {
	s := newLiveServer(t)
	f := &oauthFixture{s: s, super: s.superAdmin(), scopeIDs: map[string]string{}}

	var api struct {
		API struct {
			ID     string `json:"id"`
			Scopes []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"scopes"`
		} `json:"api"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/apis", map[string]any{
		"name": "Orders", "identifier": ordersAPI, "enforce_roles": true, "allow_offline_access": true,
		"scopes": []map[string]any{{"name": "orders:read"}, {"name": "orders:write"}},
	}, &api)
	f.apiID = api.API.ID
	for _, scope := range api.API.Scopes {
		f.scopeIDs[scope.Name] = scope.ID
	}

	f.clientID, f.clientSecret, f.appID = f.register(map[string]any{
		"name": "Shop", "type": "web",
		"grant_types":               []string{"authorization_code", "refresh_token"},
		"redirect_uris":             []string{shopRedirect},
		"post_logout_redirect_uris": []string{shopLoggedIn},
		"scopes":                    []string{"openid", "profile", "email", "offline_access", "roles"},
	})
	f.authorizeAPI(f.appID, "orders:read", "orders:write")

	var role struct {
		Role idOnly `json:"role"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/user-roles", map[string]any{
		"name": "customer", "application_id": f.appID, "api_scopes": []string{f.scopeIDs["orders:read"]},
	}, &role)

	var user struct {
		User idOnly `json:"user"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": adaEmail, "first_name": "Ada", "last_name": "Lovelace",
		"password": adaPassword, "confirm_password": adaPassword, "is_email_verified": true,
	}, &user)
	f.userID = user.User.ID

	f.super.must(http.StatusOK, http.MethodPost, "/users/"+f.userID+"/role-mappings", map[string]any{
		"roles": []string{role.Role.ID},
	}, nil)

	return f
}

// register registers an application and returns its client id, secret and id.
func (f *oauthFixture) register(body map[string]any) (clientID, secret, id string) {
	var out struct {
		Application struct {
			ID       string `json:"id"`
			ClientID string `json:"client_id"`
		} `json:"application"`
		ClientSecret string `json:"client_secret"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/applications", body, &out)

	return out.Application.ClientID, out.ClientSecret, out.Application.ID
}

func (f *oauthFixture) authorizeAPI(appID string, scopes ...string) {
	ids := []string{}
	for _, name := range scopes {
		ids = append(ids, f.scopeIDs[name])
	}
	f.super.must(http.StatusOK, http.MethodPut, "/applications/"+appID+"/apis/"+f.apiID, map[string]any{"scopes": ids}, nil)
}

// browser keeps cookies and stops at every redirect, so a test can read it.
type browser struct {
	t    *testing.T
	s    *liveServer
	http *http.Client
}

func (s *liveServer) browser() *browser {
	jar, err := cookiejar.New(nil)
	if err != nil {
		s.t.Fatal(err)
	}

	return &browser{t: s.t, s: s, http: &http.Client{
		Jar:           jar,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

// visit sends the browser to a URL and returns where it was redirected.
func (b *browser) visit(target string) *url.URL {
	b.t.Helper()

	res, err := b.http.Get(target)
	if err != nil {
		b.t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(res.Body)
		b.t.Fatalf("GET %s = %d %s, want a redirect", target, res.StatusCode, body)
	}

	location, err := url.Parse(res.Header.Get("Location"))
	if err != nil {
		b.t.Fatal(err)
	}

	return location
}

// account calls an account endpoint as the sign-in page would, from this
// browser, and decodes the answer.
func (b *browser) account(method, path string, body any, out any) int {
	b.t.Helper()

	encoded, _ := json.Marshal(body)
	req, err := http.NewRequest(method, b.s.root+"/api/v1/account"+path, bytes.NewReader(encoded))
	if err != nil {
		b.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := b.http.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer res.Body.Close()

	if out != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}

	return res.StatusCode
}

// pkce is a verifier and its S256 challenge.
func pkce(t *testing.T) (verifier, challenge string) {
	verifier = randomHex(t, 32)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *oauthFixture) authorizeURL(challenge string, extra url.Values) string {
	query := url.Values{
		"response_type":         {"code"},
		"client_id":             {f.clientID},
		"redirect_uri":          {shopRedirect},
		"scope":                 {"openid profile email offline_access roles orders:read orders:write"},
		"audience":              {ordersAPI},
		"state":                 {"state-123"},
		"nonce":                 {"nonce-456"},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	maps.Copy(query, extra)

	return f.s.root + "/oauth2/authorize?" + query.Encode()
}

// signIn runs the authorization flow up to the code: to the sign-in page, the
// password, and back to the application.
func (f *oauthFixture) signIn(b *browser, challenge string) *url.URL {
	f.s.t.Helper()
	return f.signInWith(b, challenge, nil)
}

// signInWith is signIn with parameters of the authorization request changed.
func (f *oauthFixture) signInWith(b *browser, challenge string, extra url.Values) *url.URL {
	f.s.t.Helper()

	login := b.visit(f.authorizeURL(challenge, extra))
	if !strings.HasPrefix(login.String(), testAccountURL+"/login?") {
		f.s.t.Fatalf("authorize sent the browser to %s, want the sign-in page", login)
	}
	handle := login.Query().Get("request")

	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	status := b.account(http.MethodPost, "/login", map[string]string{"request": handle, "email": adaEmail, "password": adaPassword}, &out)
	if status != http.StatusOK || out.RedirectTo == "" {
		f.s.t.Fatalf("login = %d %+v", status, out)
	}

	back, err := url.Parse(out.RedirectTo)
	if err != nil {
		f.s.t.Fatal(err)
	}

	return back
}

type tokenResult struct {
	status int
	body   map[string]any
}

func (r tokenResult) str(key string) string {
	value, _ := r.body[key].(string)
	return value
}

// token calls a form endpoint as the application: HTTP Basic when a secret is
// given, the client id in the form otherwise.
func (f *oauthFixture) token(path, clientID, secret string, form url.Values) tokenResult {
	f.s.t.Helper()

	req, err := http.NewRequest(http.MethodPost, f.s.root+path, strings.NewReader(form.Encode()))
	if err != nil {
		f.s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if secret != "" {
		req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(secret))
	} else {
		form.Set("client_id", clientID)
		req.Body = io.NopCloser(strings.NewReader(form.Encode()))
		req.ContentLength = int64(len(form.Encode()))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		f.s.t.Fatal(err)
	}
	defer res.Body.Close()

	result := tokenResult{status: res.StatusCode, body: map[string]any{}}
	_ = json.NewDecoder(res.Body).Decode(&result.body)

	return result
}

func (f *oauthFixture) exchange(code, verifier string) tokenResult {
	return f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {shopRedirect},
		"code_verifier": {verifier},
	})
}

// verifyJWT checks a token's signature against the published JWKS, the way an
// API would, and returns its header and claims.
func (f *oauthFixture) verifyJWT(token string) (map[string]any, map[string]any) {
	t := f.s.t
	t.Helper()

	res, err := http.Get(f.s.root + "/.well-known/jwks.json")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var jwks struct {
		Keys []map[string]string `json:"keys"`
	}
	if err := json.NewDecoder(res.Body).Decode(&jwks); err != nil {
		t.Fatal(err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("not a JWT: %q", token)
	}

	decode := func(part string) map[string]any {
		raw, err := base64.RawURLEncoding.DecodeString(part)
		if err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	header, claims := decode(parts[0]), decode(parts[1])

	for _, key := range jwks.Keys {
		if key["kid"] != header["kid"] {
			continue
		}
		if key["kty"] != "RSA" || header["alg"] != "RS256" {
			t.Fatalf("test only checks RS256; got key %v and header %v", key, header)
		}

		n, _ := base64.RawURLEncoding.DecodeString(key["n"])
		e, _ := base64.RawURLEncoding.DecodeString(key["e"])
		public := &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}

		signature, _ := base64.RawURLEncoding.DecodeString(parts[2])
		digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
		if err := rsa.VerifyPKCS1v15(public, crypto.SHA256, digest[:], signature); err != nil {
			t.Fatalf("signature does not verify against the JWKS: %v", err)
		}

		return header, claims
	}

	t.Fatalf("no key %v in the JWKS", header["kid"])
	return nil, nil
}

func stringList(values any) []string {
	out := []string{}
	list, _ := values.([]any)
	for _, v := range list {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func TestLiveOAuthAuthorizationCodeFlow(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	// Discovery names this server.
	res, err := http.Get(f.s.root + "/.well-known/openid-configuration")
	if err != nil {
		t.Fatal(err)
	}
	var discovery map[string]any
	_ = json.NewDecoder(res.Body).Decode(&discovery)
	res.Body.Close()
	if discovery["issuer"] != f.s.root || discovery["token_endpoint"] != f.s.root+"/oauth2/token" {
		t.Fatalf("discovery = %v", discovery)
	}

	// The sign-in page learns which application it is for.
	verifier, challenge := pkce(t)
	login := b.visit(f.authorizeURL(challenge, nil))
	handle := login.Query().Get("request")

	var pending struct {
		Application struct {
			Name              string `json:"name"`
			AllowRegistration bool   `json:"allow_registration"`
		} `json:"application"`
	}
	if status := b.account(http.MethodGet, "/requests/"+handle, nil, &pending); status != http.StatusOK || pending.Application.Name != "Shop" || !pending.Application.AllowRegistration {
		t.Fatalf("request = %d %+v", status, pending)
	}

	if status := b.account(http.MethodPost, "/login", map[string]string{"request": handle, "email": adaEmail, "password": "wrong-password"}, nil); status != http.StatusUnauthorized {
		t.Errorf("wrong password = %d, want 401", status)
	}

	var signedIn struct {
		RedirectTo string `json:"redirect_to"`
	}
	b.account(http.MethodPost, "/login", map[string]string{"request": handle, "email": adaEmail, "password": adaPassword}, &signedIn)
	back, _ := url.Parse(signedIn.RedirectTo)

	if got := back.Scheme + "://" + back.Host + back.Path; got != shopRedirect {
		t.Fatalf("redirected to %s, want the application's redirect URI", signedIn.RedirectTo)
	}
	if back.Query().Get("state") != "state-123" || back.Query().Get("iss") != f.s.root || back.Query().Get("code") == "" {
		t.Fatalf("callback query = %v", back.Query())
	}

	// The handle is spent.
	if status := b.account(http.MethodGet, "/requests/"+handle, nil, nil); status != http.StatusGone {
		t.Errorf("used handle = %d, want 410", status)
	}

	code := back.Query().Get("code")
	tokens := f.exchange(code, verifier)
	if tokens.status != http.StatusOK {
		t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
	}

	scope := strings.Fields(tokens.str("scope"))
	if !slices.Contains(scope, "orders:read") || slices.Contains(scope, "orders:write") {
		t.Errorf("scope = %v, want orders:read granted by the role and orders:write refused", scope)
	}

	accessHeader, access := f.verifyJWT(tokens.str("access_token"))
	if accessHeader["typ"] != "at+jwt" || access["iss"] != f.s.root || access["sub"] != f.userID || access["client_id"] != f.clientID {
		t.Errorf("access token = %v %v", accessHeader, access)
	}
	if aud := stringList(access["aud"]); len(aud) != 1 || aud[0] != ordersAPI {
		t.Errorf("access aud = %v", access["aud"])
	}

	_, id := f.verifyJWT(tokens.str("id_token"))
	if id["aud"] != f.clientID || id["nonce"] != "nonce-456" || id["email"] != adaEmail || id["given_name"] != "Ada" || id["at_hash"] == nil || id["auth_time"] == nil {
		t.Errorf("id token = %v", id)
	}
	if roles := stringList(id["roles"]); len(roles) != 1 || roles[0] != "customer" {
		t.Errorf("id token roles = %v", id["roles"])
	}

	// Userinfo describes the user from the access token.
	req, _ := http.NewRequest(http.MethodGet, f.s.root+"/oauth2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.str("access_token"))
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var info map[string]any
	_ = json.NewDecoder(res.Body).Decode(&info)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || info["sub"] != f.userID || info["email"] != adaEmail || info["iss"] != nil {
		t.Errorf("userinfo = %d %v", res.StatusCode, info)
	}

	req, _ = http.NewRequest(http.MethodGet, f.s.root+"/oauth2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tokens.str("id_token"))
	res, _ = http.DefaultClient.Do(req)
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized || !strings.Contains(res.Header.Get("WWW-Authenticate"), "invalid_token") {
		t.Errorf("userinfo with an ID token = %d %q, want 401 invalid_token", res.StatusCode, res.Header.Get("WWW-Authenticate"))
	}

	// Refreshing rotates the token; the old one presented again revokes the
	// family, the new one included.
	first := tokens.str("refresh_token")
	refreshed := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {first}})
	if refreshed.status != http.StatusOK || refreshed.str("refresh_token") == "" || refreshed.str("refresh_token") == first {
		t.Fatalf("refresh = %d %v", refreshed.status, refreshed.body)
	}

	replay := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {first}})
	if replay.status != http.StatusBadRequest || replay.str("error") != "invalid_grant" {
		t.Errorf("replayed refresh token = %d %v", replay.status, replay.body)
	}
	if again := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshed.str("refresh_token")}}); again.status != http.StatusBadRequest {
		t.Errorf("refresh after a replay = %d %v, want the family revoked", again.status, again.body)
	}

	// The same code a second time is refused.
	if reused := f.exchange(code, verifier); reused.str("error") != "invalid_grant" {
		t.Errorf("reused code = %d %v", reused.status, reused.body)
	}

	// Signed in already: a second authorization skips the sign-in page.
	_, challenge = pkce(t)
	straight := b.visit(f.authorizeURL(challenge, nil))
	if !strings.HasPrefix(straight.String(), shopRedirect+"?") || straight.Query().Get("code") == "" {
		t.Fatalf("second authorize went to %s, want straight back with a code", straight)
	}

	if wrong := f.exchange(straight.Query().Get("code"), strings.Repeat("x", 43)); wrong.str("error") != "invalid_grant" {
		t.Errorf("wrong code_verifier = %d %v", wrong.status, wrong.body)
	}

	// prompt=login asks again even so.
	if again := b.visit(f.authorizeURL(challenge, url.Values{"prompt": {"login"}})); !strings.HasPrefix(again.String(), testAccountURL) {
		t.Errorf("prompt=login went to %s, want the sign-in page", again)
	}

	// Signing out with the ID token returns to the registered URI, and ends
	// the session: the next authorization asks for the password.
	logout := b.visit(f.s.root + "/oauth2/logout?" + url.Values{
		"id_token_hint":            {tokens.str("id_token")},
		"post_logout_redirect_uri": {shopLoggedIn},
		"state":                    {"bye"},
	}.Encode())
	if logout.String() != shopLoggedIn+"?state=bye" {
		t.Errorf("logout redirected to %s", logout)
	}

	if after := b.visit(f.authorizeURL(challenge, url.Values{"prompt": {"none"}})); after.Query().Get("error") != "login_required" {
		t.Errorf("prompt=none after logout = %s, want login_required", after)
	}

	// A logout naming a URI the application did not register goes nowhere.
	if bad := b.visit(f.s.root + "/oauth2/logout?" + url.Values{"client_id": {f.clientID}, "post_logout_redirect_uri": {"https://evil.example.com"}}.Encode()); !strings.HasPrefix(bad.String(), testAccountURL+"/error") {
		t.Errorf("unregistered post-logout URI went to %s", bad)
	}
}

// A refresh that narrows the scope past offline_access still spends the token
// it was given. Leaving it usable would be a way to hold a stolen refresh
// token for ever: a fresh access token every time, and the reuse detection
// never reached.
func TestLiveOAuthRefreshRotatesEvenWhenScopeNarrows(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	verifier, challenge := pkce(t)
	back := f.signIn(b, challenge)

	tokens := f.exchange(back.Query().Get("code"), verifier)
	first := tokens.str("refresh_token")
	if tokens.status != http.StatusOK || first == "" {
		t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
	}

	// openid alone: a subset of what was granted, and without offline_access.
	narrowed := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {first},
		"scope":         {"openid"},
	})
	if narrowed.status != http.StatusOK {
		t.Fatalf("narrowed refresh = %d %v", narrowed.status, narrowed.body)
	}
	if scope := strings.Fields(narrowed.str("scope")); slices.Contains(scope, "offline_access") {
		t.Fatalf("scope = %v, want offline_access narrowed away", scope)
	}

	next := narrowed.str("refresh_token")
	if next == "" || next == first {
		t.Fatalf("narrowed refresh returned %q, want a rotated token", next)
	}

	// The token it replaced is spent, and presenting it again takes the family
	// with it — the new one included.
	replay := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {first},
	})
	if replay.status != http.StatusBadRequest || replay.str("error") != "invalid_grant" {
		t.Errorf("replayed refresh token = %d %v, want invalid_grant", replay.status, replay.body)
	}

	after := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {next},
	})
	if after.status != http.StatusBadRequest {
		t.Errorf("refresh after a replay = %d %v, want the family revoked", after.status, after.body)
	}
}

// A code is spent by the client it was issued to and by nobody else — and an
// attempt by another client leaves it as it was, rather than burning it. A code
// can leak into a Referer header, a proxy log or an open redirect, and anyone
// holding one could otherwise deny the sign-in with a single request.
func TestLiveOAuthAnotherClientCannotSpendACode(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	rivalID, rivalSecret, _ := f.register(map[string]any{
		"name": "Rival", "type": "web",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"https://rival.example.com/callback"},
		"scopes":        []string{"openid"},
	})

	verifier, challenge := pkce(t)
	code := f.signIn(b, challenge).Query().Get("code")
	if code == "" {
		t.Fatal("no code came back from the sign-in")
	}

	stolen := f.token("/oauth2/token", rivalID, rivalSecret, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {shopRedirect},
		"code_verifier": {verifier},
	})
	if stolen.status != http.StatusBadRequest || stolen.str("error") != "invalid_grant" {
		t.Fatalf("another client's exchange = %d %v, want invalid_grant", stolen.status, stolen.body)
	}

	// Untouched: the client it belongs to still gets its tokens.
	tokens := f.exchange(code, verifier)
	if tokens.status != http.StatusOK || tokens.str("access_token") == "" {
		t.Fatalf("exchange after another client tried = %d %v", tokens.status, tokens.body)
	}
}

// The logout endpoint is a plain GET, so anybody can put it in a link and get
// somebody else to follow it. A request that was refused, or that names
// another user, must leave the reader signed in — cookie included.
func TestLiveOAuthLogoutLeavesOtherPeoplesSessionsAlone(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	verifier, challenge := pkce(t)
	tokens := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	if tokens.status != http.StatusOK {
		t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
	}

	signedIn := func() bool {
		return b.account(http.MethodGet, "/me", nil, nil) == http.StatusOK
	}
	if !signedIn() {
		t.Fatal("the browser is not signed in to start with")
	}

	// A hint this server never signed is refused, and the reader stays in.
	refused := b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {"not.a.token"}}.Encode())
	if !strings.HasPrefix(refused.String(), testAccountURL+"/error") {
		t.Errorf("a made-up hint led to %s, want the error page", refused)
	}
	if !signedIn() {
		t.Fatal("a made-up hint signed the reader out")
	}

	// Somebody else's ID token, signed by this server and perfectly valid, is
	// not about this browser either.
	stranger := f.s.browser()
	strangerVerifier, strangerChallenge := pkce(t)
	f.super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "grace@example.com", "first_name": "Grace",
		"password": "grace-password-1", "confirm_password": "grace-password-1",
		"is_email_verified": true,
	}, nil)

	login := stranger.visit(f.authorizeURL(strangerChallenge, nil))
	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	stranger.account(http.MethodPost, "/login", map[string]string{
		"request": login.Query().Get("request"), "email": "grace@example.com", "password": "grace-password-1",
	}, &out)
	back, _ := url.Parse(out.RedirectTo)
	theirs := f.exchange(back.Query().Get("code"), strangerVerifier)
	if theirs.str("id_token") == "" {
		t.Fatalf("the stranger got no ID token: %v", theirs.body)
	}

	b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {theirs.str("id_token")}}.Encode())
	if !signedIn() {
		t.Fatal("a logout naming another user signed this browser out")
	}

	// The reader's own ID token still signs the reader out.
	b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {tokens.str("id_token")}}.Encode())
	if signedIn() {
		t.Error("a logout with the reader's own ID token left them signed in")
	}
}

// A logout without id_token_hint could come from any page that links to it,
// so it asks the reader first (RP-Initiated Logout §2). Only the confirmation
// page's own JSON call, which another site cannot make, signs them out.
func TestLiveOAuthLogoutWithoutAHintAsksFirst(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	verifier, challenge := pkce(t)
	if tokens := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier); tokens.status != http.StatusOK {
		t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
	}

	signedIn := func() bool {
		return b.account(http.MethodGet, "/me", nil, nil) == http.StatusOK
	}

	logout := url.Values{"client_id": {f.clientID}, "post_logout_redirect_uri": {shopLoggedIn}, "state": {"bye"}}

	for name, landed := range map[string]*url.URL{
		"a link":        b.visit(f.s.root + "/oauth2/logout?" + logout.Encode()),
		"a posted form": b.post(t, f.s.root+"/oauth2/logout", logout).landed,
	} {
		if !strings.HasPrefix(landed.String(), testAccountURL+"/logout?") {
			t.Errorf("%s led to %s, want the confirmation page", name, landed)
		}
		for key, want := range logout {
			if got := landed.Query().Get(key); got != want[0] {
				t.Errorf("%s: the confirmation page was given %s=%q, want %q", name, key, got, want[0])
			}
		}
		if !signedIn() {
			t.Fatalf("%s signed the reader out without asking", name)
		}
	}

	// The confirmation is checked like the request: an address the
	// application did not register goes to the error page, still signed in.
	var answer struct {
		Location string `json:"location"`
	}
	b.account(http.MethodPost, "/logout", map[string]string{
		"client_id": f.clientID, "post_logout_redirect_uri": "https://evil.example.com",
	}, &answer)
	if !strings.HasPrefix(answer.Location, testAccountURL+"/error") {
		t.Errorf("an unregistered address confirmed led to %s", answer.Location)
	}
	if !signedIn() {
		t.Fatal("a refused confirmation signed the reader out")
	}

	if status := b.account(http.MethodPost, "/logout", map[string]string{
		"client_id": f.clientID, "post_logout_redirect_uri": shopLoggedIn, "state": "bye",
	}, &answer); status != http.StatusOK || answer.Location != shopLoggedIn+"?state=bye" {
		t.Errorf("confirming = %d, location %s", status, answer.Location)
	}
	if signedIn() {
		t.Fatal("confirming left the reader signed in")
	}

	// Nobody to ask: a browser that is not signed in goes straight on.
	if landed := b.visit(f.s.root + "/oauth2/logout?" + logout.Encode()); landed.String() != shopLoggedIn+"?state=bye" {
		t.Errorf("a signed-out browser was sent to %s", landed)
	}
}

func TestLiveOAuthRefusesUntrustedRequests(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()
	_, challenge := pkce(t)

	// A redirect URI the application did not register never gets the error.
	evil := b.visit(f.authorizeURL(challenge, url.Values{"redirect_uri": {"https://evil.example.com/callback"}}))
	if !strings.HasPrefix(evil.String(), testAccountURL+"/error?") {
		t.Errorf("unregistered redirect_uri went to %s", evil)
	}

	// PKCE is required for this application.
	noPKCE := b.visit(f.s.root + "/oauth2/authorize?" + url.Values{
		"response_type": {"code"}, "client_id": {f.clientID}, "redirect_uri": {shopRedirect}, "scope": {"openid"}, "state": {"s"},
	}.Encode())
	if noPKCE.Query().Get("error") != "invalid_request" || noPKCE.Query().Get("state") != "s" {
		t.Errorf("missing code_challenge = %s", noPKCE)
	}

	// The token endpoint holds the client to its registered method.
	post := f.token("/oauth2/token", f.clientID, "", url.Values{"grant_type": {"authorization_code"}, "client_secret": {f.clientSecret}, "code": {"x"}})
	if post.status != http.StatusUnauthorized || post.str("error") != "invalid_client" {
		t.Errorf("client_secret_post for a basic client = %d %v", post.status, post.body)
	}

	wrong := f.token("/oauth2/token", f.clientID, "not-the-secret", url.Values{"grant_type": {"authorization_code"}, "code": {"x"}})
	if wrong.status != http.StatusUnauthorized {
		t.Errorf("wrong secret = %d %v", wrong.status, wrong.body)
	}

	// A JSON body is named as the mistake it is.
	req, _ := http.NewRequest(http.MethodPost, f.s.root+"/oauth2/token", strings.NewReader(`{"grant_type":"client_credentials"}`))
	req.Header.Set("Content-Type", "application/json")
	res, _ := http.DefaultClient.Do(req)
	var body map[string]string
	_ = json.NewDecoder(res.Body).Decode(&body)
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest || !strings.Contains(body["error_description"], "x-www-form-urlencoded") {
		t.Errorf("JSON token request = %d %v", res.StatusCode, body)
	}

	// A body far larger than anything this server takes is refused outright,
	// rather than read into memory first.
	huge, err := http.NewRequest(http.MethodPost, f.s.root+"/api/v1/account/login", strings.NewReader(strings.Repeat("a", 2<<20)))
	if err != nil {
		t.Fatal(err)
	}
	huge.Header.Set("Content-Type", "application/json")

	oversized, err := http.DefaultClient.Do(huge)
	if err != nil {
		t.Fatal(err)
	}
	oversized.Body.Close()
	if oversized.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("a two-megabyte body = %d, want 413", oversized.StatusCode)
	}

	// A user without the application's required role is turned away at
	// authorization, back to the application with access_denied.
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "grant_types": []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{shopRedirect}, "post_logout_redirect_uris": []string{shopLoggedIn},
		"scopes": []string{"openid", "profile", "email", "offline_access", "roles"}, "require_role_assignment": true,
	}, nil)

	var user struct {
		User idOnly `json:"user"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "norole@example.com", "password": "norole-password", "confirm_password": "norole-password",
	}, &user)

	login := b.visit(f.authorizeURL(challenge, nil))
	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	b.account(http.MethodPost, "/login", map[string]string{"request": login.Query().Get("request"), "email": "norole@example.com", "password": "norole-password"}, &out)
	denied, _ := url.Parse(out.RedirectTo)
	if denied.Query().Get("error") != "access_denied" {
		t.Errorf("user without a required role = %s, want access_denied", out.RedirectTo)
	}
}

func TestLiveOAuthClientCredentialsRevokeAndIntrospect(t *testing.T) {
	f := newOAuthFixture(t)

	clientID, secret, appID := f.register(map[string]any{"name": "Nightly job", "type": "m2m"})
	f.authorizeAPI(appID, "orders:read")

	tokens := f.token("/oauth2/token", clientID, secret, url.Values{
		"grant_type": {"client_credentials"}, "audience": {ordersAPI}, "scope": {"orders:read orders:write openid"},
	})
	if tokens.status != http.StatusOK || tokens.str("scope") != "orders:read" || tokens.str("refresh_token") != "" || tokens.str("id_token") != "" {
		t.Fatalf("client credentials = %d %v", tokens.status, tokens.body)
	}

	_, access := f.verifyJWT(tokens.str("access_token"))
	if access["sub"] != clientID {
		t.Errorf("m2m sub = %v, want the client id", access["sub"])
	}

	// A web application's token, introspected and then revoked.
	b := f.s.browser()
	verifier, challenge := pkce(t)
	web := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)

	introspected := f.token("/oauth2/introspect", f.clientID, f.clientSecret, url.Values{"token": {web.str("access_token")}})
	if introspected.body["active"] != true || introspected.body["sub"] != f.userID {
		t.Errorf("introspect access token = %v", introspected.body)
	}

	if refresh := f.token("/oauth2/introspect", f.clientID, f.clientSecret, url.Values{"token": {web.str("refresh_token")}}); refresh.body["active"] != true {
		t.Errorf("introspect refresh token = %v", refresh.body)
	}

	// Another client may not see or revoke it. A signed access token verifies
	// for anyone, so this is the only thing keeping one client from reading
	// the subject, the scope and the roles out of another client's token.
	if other := f.token("/oauth2/introspect", clientID, secret, url.Values{"token": {web.str("access_token")}}); other.body["active"] != false || other.body["sub"] != nil {
		t.Errorf("another client's introspection of an access token = %v", other.body)
	}

	if other := f.token("/oauth2/introspect", clientID, secret, url.Values{"token": {web.str("refresh_token")}}); other.body["active"] != false {
		t.Errorf("another client's introspection of a refresh token = %v", other.body)
	}

	// Its own token it may still read: the check is on whose token it is, not
	// on introspection itself.
	if own := f.token("/oauth2/introspect", clientID, secret, url.Values{"token": {tokens.str("access_token")}}); own.body["active"] != true || own.body["sub"] != clientID {
		t.Errorf("a client's introspection of its own token = %v", own.body)
	}

	if revoked := f.token("/oauth2/revoke", f.clientID, f.clientSecret, url.Values{"token": {web.str("refresh_token")}}); revoked.status != http.StatusOK {
		t.Fatalf("revoke = %d %v", revoked.status, revoked.body)
	}

	if after := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {web.str("refresh_token")}}); after.str("error") != "invalid_grant" {
		t.Errorf("refresh after revoke = %d %v", after.status, after.body)
	}
}

// An API that introspects is told what userinfo and the refresh grant already
// say: a user turned off or deleted since the token was issued is not active.
func TestLiveOAuthIntrospectionForgetsUsersWhoAreGone(t *testing.T) {
	tests := []struct {
		name   string
		remove func(f *oauthFixture)
	}{
		{"a user turned off", func(f *oauthFixture) {
			f.super.must(http.StatusOK, http.MethodPatch, "/users/"+f.userID, map[string]any{
				"email": adaEmail, "first_name": "Ada", "last_name": "Lovelace", "is_active": false,
			}, nil)
		}},
		{"a user deleted", func(f *oauthFixture) {
			f.super.must(http.StatusNoContent, http.MethodDelete, "/users/"+f.userID, nil, nil)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			verifier, challenge := pkce(t)
			web := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier)

			if before := f.token("/oauth2/introspect", f.clientID, f.clientSecret, url.Values{"token": {web.str("access_token")}}); before.body["active"] != true {
				t.Fatalf("introspect before = %v", before.body)
			}

			tt.remove(f)

			after := f.token("/oauth2/introspect", f.clientID, f.clientSecret, url.Values{"token": {web.str("access_token")}})
			if after.body["active"] != false || len(after.body) != 1 {
				t.Errorf("introspect after = %v, want only active=false", after.body)
			}
		})
	}
}

// Every sign-in writes the User-Agent to the session row, cut to fit the
// column. The header is whatever the caller sent, so the cut has to leave
// something the database will take: through the middle of a two-byte letter it
// does not, and the sign-in fails rather than the name it was carrying.
func TestLiveAccountSignsInWithAnAwkwardUserAgent(t *testing.T) {
	f := newOAuthFixture(t)

	body, err := json.Marshal(map[string]string{"email": adaEmail, "password": adaPassword})
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, f.s.root+"/api/v1/account/login", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Longer than the column, in letters of two bytes each — so the limit
	// falls inside one — and with a byte that is not UTF-8 at all.
	req.Header.Set("User-Agent", strings.Repeat("\u044f", 200)+"\xff")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		answer, _ := io.ReadAll(res.Body)
		t.Fatalf("sign-in with an awkward User-Agent = %d %s, want 200", res.StatusCode, answer)
	}
}

func TestLiveAccountRegisterAndResetPassword(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()
	verifier, challenge := pkce(t)

	login := b.visit(f.authorizeURL(challenge, nil))
	handle := login.Query().Get("request")

	if status := b.account(http.MethodPost, "/register", map[string]any{"request": handle, "email": adaEmail, "password": "another-password"}, nil); status != http.StatusConflict {
		t.Errorf("registering a taken email = %d, want 409", status)
	}

	var out struct {
		RedirectTo string `json:"redirect_to"`
	}
	status := b.account(http.MethodPost, "/register", map[string]any{
		"request": handle, "email": "Grace@Example.com", "password": "grace-password", "first_name": "Grace",
	}, &out)
	back, _ := url.Parse(out.RedirectTo)
	if status != http.StatusOK || back.Query().Get("code") == "" {
		t.Fatalf("register = %d %+v", status, out)
	}

	tokens := f.exchange(back.Query().Get("code"), verifier)
	_, id := f.verifyJWT(tokens.str("id_token"))
	if id["email"] != "grace@example.com" || id["email_verified"] != false {
		t.Errorf("registered user's id token = %v", id)
	}

	// Terms have to be accepted once the application links them.
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "grant_types": []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{shopRedirect}, "scopes": []string{"openid", "email"},
		"terms_url": "https://shop.example.com/terms", "privacy_url": "https://shop.example.com/privacy",
	}, nil)

	_, challenge = pkce(t)
	handle = f.s.browser().visit(f.authorizeURL(challenge, nil)).Query().Get("request")

	var pending struct {
		Application struct {
			TermsURL string `json:"terms_url"`
		} `json:"application"`
	}
	b.account(http.MethodGet, "/requests/"+handle, nil, &pending)
	if pending.Application.TermsURL != "https://shop.example.com/terms" {
		t.Errorf("request terms_url = %q", pending.Application.TermsURL)
	}

	if status := b.account(http.MethodPost, "/register", map[string]any{"request": handle, "email": "lin@example.com", "password": "lin-password"}, nil); status != http.StatusBadRequest {
		t.Errorf("registering without accepting terms = %d, want 400", status)
	}

	// Forgot password: the same answer for an unknown address, and a link
	// by email for a real one.
	if status := b.account(http.MethodPost, "/forgot-password", map[string]string{"email": "nobody@example.com"}, nil); status != http.StatusAccepted {
		t.Errorf("forgot password for an unknown address = %d, want 202", status)
	}
	if status := b.account(http.MethodPost, "/forgot-password", map[string]string{"email": adaEmail, "request": handle}, nil); status != http.StatusAccepted {
		t.Fatalf("forgot password = %d", status)
	}

	msg := f.s.mail.wait(t, adaEmail)
	link := linkIn(t, msg.Body, "/reset-password")
	token := link.Query().Get("token")
	if link.Query().Get("request") != handle {
		t.Errorf("reset link %s does not carry the sign-in request back", link)
	}

	var check struct {
		Valid bool `json:"valid"`
	}
	b.account(http.MethodGet, "/reset-password?token="+url.QueryEscape(token), nil, &check)
	if !check.Valid {
		t.Error("a fresh reset link is not valid")
	}

	const newPassword = "ada-new-password"
	if status := b.account(http.MethodPost, "/reset-password", map[string]string{"token": token, "password": newPassword}, nil); status != http.StatusOK {
		t.Fatalf("reset = %d", status)
	}
	if status := b.account(http.MethodPost, "/reset-password", map[string]string{"token": token, "password": "yet-another-one"}, nil); status != http.StatusGone {
		t.Errorf("reset link used twice = %d, want 410", status)
	}

	if status := b.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": adaPassword}, nil); status != http.StatusUnauthorized {
		t.Errorf("old password after reset = %d, want 401", status)
	}
	if status := b.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": newPassword}, nil); status != http.StatusOK {
		t.Errorf("new password after reset = %d, want 200", status)
	}

	// A temporary password an administrator set sends the user to choose one.
	f.super.must(http.StatusOK, http.MethodPatch, "/users/"+f.userID, map[string]any{
		"email": adaEmail, "password": "temporary-1", "confirm_password": "temporary-1", "is_password_temporary": true,
	}, nil)

	var temporary struct {
		PasswordChangeRequired bool   `json:"password_change_required"`
		ResetToken             string `json:"reset_token"`
	}
	b.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": "temporary-1"}, &temporary)
	if !temporary.PasswordChangeRequired || temporary.ResetToken == "" {
		t.Errorf("temporary password login = %+v", temporary)
	}

	// Registration switched off.
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "grant_types": []string{"authorization_code"},
		"redirect_uris": []string{shopRedirect}, "scopes": []string{"openid"}, "allow_registration": false,
	}, nil)
	handle = f.s.browser().visit(f.authorizeURL(challenge, nil)).Query().Get("request")
	if status := b.account(http.MethodPost, "/register", map[string]any{"request": handle, "email": "closed@example.com", "password": "closed-password", "accept_terms": true}, nil); status != http.StatusForbidden {
		t.Errorf("registering with registration off = %d, want 403", status)
	}
}

func TestLiveAccountManagement(t *testing.T) {
	f := newOAuthFixture(t)

	// Nothing without a session.
	stranger := f.s.browser()
	if status := stranger.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Fatalf("me without a session = %d, want 401", status)
	}

	// Signed in twice: on a laptop, through an application, and on a phone.
	laptop := f.s.browser()
	verifier, challenge := pkce(t)
	tokens := f.exchange(f.signIn(laptop, challenge).Query().Get("code"), verifier)
	if tokens.str("refresh_token") == "" {
		t.Fatalf("no refresh token to disconnect: %v", tokens.body)
	}

	phone := f.s.browser()
	if status := phone.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": adaPassword}, nil); status != http.StatusOK {
		t.Fatalf("phone login = %d", status)
	}

	var me struct {
		User map[string]any `json:"user"`
	}
	laptop.account(http.MethodGet, "/me", nil, &me)
	if me.User["email"] != adaEmail || me.User["roles"] != nil || me.User["password_hash"] != nil {
		t.Errorf("me = %v", me.User)
	}

	laptop.account(http.MethodPatch, "/me", map[string]string{"first_name": " Augusta ", "last_name": "King"}, &me)
	if me.User["first_name"] != "Augusta" || me.User["last_name"] != "King" {
		t.Errorf("updated me = %v", me.User)
	}

	// The laptop sees both sessions, and signs the phone out.
	var sessions struct {
		Sessions []struct {
			ID      string `json:"id"`
			Current bool   `json:"current"`
		} `json:"sessions"`
	}
	laptop.account(http.MethodGet, "/sessions", nil, &sessions)
	if len(sessions.Sessions) != 2 {
		t.Fatalf("sessions = %+v, want the laptop's and the phone's", sessions.Sessions)
	}

	var phoneSession, laptopSession string
	for _, s := range sessions.Sessions {
		if s.Current {
			laptopSession = s.ID
		} else {
			phoneSession = s.ID
		}
	}

	if status := laptop.account(http.MethodDelete, "/sessions/"+laptopSession, nil, nil); status != http.StatusBadRequest {
		t.Errorf("ending the current session = %d, want 400", status)
	}
	if status := laptop.account(http.MethodDelete, "/sessions/"+phoneSession, nil, nil); status != http.StatusNoContent {
		t.Fatalf("ending the phone's session = %d", status)
	}
	if status := phone.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("phone after being signed out = %d, want 401", status)
	}

	// Someone else's session cannot be ended.
	other := f.s.browser()
	var registered struct {
		RedirectTo string `json:"redirect_to"`
	}
	handle := other.visit(f.authorizeURL(challenge, nil)).Query().Get("request")
	other.account(http.MethodPost, "/register", map[string]any{"request": handle, "email": "mallory@example.com", "password": "mallory-password"}, &registered)
	if status := other.account(http.MethodDelete, "/sessions/"+laptopSession, nil, nil); status != http.StatusNotFound {
		t.Errorf("ending another user's session = %d, want 404", status)
	}

	// The application that signed in is connected, and can be disconnected.
	var connected struct {
		Applications []struct {
			ClientID string   `json:"client_id"`
			Name     string   `json:"name"`
			Scopes   []string `json:"scopes"`
		} `json:"applications"`
	}
	laptop.account(http.MethodGet, "/connected-applications", nil, &connected)
	if len(connected.Applications) != 1 || connected.Applications[0].Name != "Shop" || !slices.Contains(connected.Applications[0].Scopes, "offline_access") {
		t.Fatalf("connected applications = %+v", connected.Applications)
	}

	if status := laptop.account(http.MethodDelete, "/connected-applications/"+f.clientID, nil, nil); status != http.StatusNoContent {
		t.Fatalf("disconnect = %d", status)
	}
	if after := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.str("refresh_token")}}); after.str("error") != "invalid_grant" {
		t.Errorf("refresh after disconnecting = %d %v", after.status, after.body)
	}
	if status := laptop.account(http.MethodDelete, "/connected-applications/"+f.clientID, nil, nil); status != http.StatusNotFound {
		t.Errorf("disconnecting twice = %d, want 404", status)
	}

	// Changing the password needs the current one, and signs out everywhere
	// else.
	if status := laptop.account(http.MethodPost, "/password", map[string]string{"current_password": "wrong-one", "new_password": "ada-changed-password"}, nil); status != http.StatusBadRequest {
		t.Errorf("wrong current password = %d, want 400", status)
	}

	tablet := f.s.browser()
	tablet.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": adaPassword}, nil)

	if status := laptop.account(http.MethodPost, "/password", map[string]string{"current_password": adaPassword, "new_password": "ada-changed-password"}, nil); status != http.StatusOK {
		t.Fatalf("change password = %d", status)
	}
	if status := laptop.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Errorf("the laptop after changing the password = %d, want still signed in", status)
	}
	if status := tablet.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("another session after a password change = %d, want 401", status)
	}
	if status := tablet.account(http.MethodPost, "/login", map[string]string{"email": adaEmail, "password": "ada-changed-password"}, nil); status != http.StatusOK {
		t.Errorf("login with the new password = %d", status)
	}
}

// A super admin rotating the keys with revocation: tokens the old keys signed
// stop verifying, and new tokens are signed by the new keys.
func TestLiveSigningKeyRevocation(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	verifier, challenge := pkce(t)
	before := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	oldHeader, _ := f.verifyJWT(before.str("access_token"))

	var listed struct {
		Keys []struct {
			KID   string `json:"kid"`
			State string `json:"state"`
		} `json:"keys"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/signing-keys", nil, &listed)
	if len(listed.Keys) != 3 {
		t.Fatalf("signing keys = %+v, want one per algorithm", listed.Keys)
	}

	f.super.must(http.StatusOK, http.MethodPost, "/signing-keys/rotate", map[string]bool{"revoke_old": true}, &listed)
	for _, key := range listed.Keys {
		if key.KID == oldHeader["kid"] {
			t.Errorf("the revoked key %v is still listed", key.KID)
		}
	}

	req, _ := http.NewRequest(http.MethodGet, f.s.root+"/oauth2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+before.str("access_token"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("userinfo with a token from a revoked key = %d, want 401", res.StatusCode)
	}

	verifier, challenge = pkce(t)
	after := f.exchange(b.visit(f.authorizeURL(challenge, nil)).Query().Get("code"), verifier)
	newHeader, _ := f.verifyJWT(after.str("access_token"))
	if newHeader["kid"] == oldHeader["kid"] {
		t.Error("a token issued after rotating is signed by the old key")
	}

	// Only a super admin may rotate.
	manager := f.s.appManager(f.super, f.appID)
	if status := manager.do(http.MethodPost, "/signing-keys/rotate", nil, nil); status != http.StatusForbidden {
		t.Errorf("rotation by an app manager = %d, want 403", status)
	}
}

// What a client credentials token carries is decided by the API it names:
// the API has to exist, the application has to be allowed it, and only the
// scopes the application was given survive.
func TestLiveOAuthAudienceDecidesTheToken(t *testing.T) {
	f := newOAuthFixture(t)

	allowed, allowedSecret, allowedID := f.register(map[string]any{"name": "Allowed job", "type": "m2m"})
	f.authorizeAPI(allowedID, "orders:read")
	stranger, strangerSecret, _ := f.register(map[string]any{"name": "Stranger job", "type": "m2m"})

	for _, tt := range []struct {
		name      string
		clientID  string
		secret    string
		audience  string
		scope     string
		wantError string
		wantScope string
	}{
		{"an API that does not exist", allowed, allowedSecret, "https://api.example.com/nothing", "", "invalid_request", ""},
		{"an API the application is not allowed", stranger, strangerSecret, ordersAPI, "orders:read", "unauthorized_client", ""},
		{"a scope the application was not given", allowed, allowedSecret, ordersAPI, "orders:read orders:write", "", "orders:read"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := f.token("/oauth2/token", tt.clientID, tt.secret, url.Values{
				"grant_type": {"client_credentials"}, "audience": {tt.audience}, "scope": {tt.scope},
			})
			if r.str("error") != tt.wantError || r.str("scope") != tt.wantScope {
				t.Errorf("= %d %v, want error %q and scope %q", r.status, r.body, tt.wantError, tt.wantScope)
			}
		})
	}
}
