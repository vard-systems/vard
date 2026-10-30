package api

import (
	"encoding/json"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Signing in with an account somewhere else, end to end.
//
// The provider it signs in with is this same server: the test registers an
// application here, points a provider of kind "oidc" at this server's own
// authorize, token and userinfo endpoints, and then walks a browser through
// the whole of it. Nothing is stubbed — the code that talks to Google is the
// code under test, and what it talks to is an OpenID Connect provider that
// happens to be the one running the test.

// socialFixture is a server with a provider configured to sign users in
// through itself, and a user to sign in as.
type socialFixture struct {
	s     *liveServer
	super *client

	slug     string
	email    string
	password string
	userID   string
}

const socialSlug = "self"

func newSocialFixture(t *testing.T, provider map[string]any) *socialFixture {
	t.Helper()

	s := newLiveServer(t)
	super := s.superAdmin()

	f := &socialFixture{
		s: s, super: super, slug: socialSlug,
		email: "user@example.com", password: "user-password-1",
	}

	// The client this server will be of itself. Its redirect URI is the
	// callback the social provider sends browsers back to.
	var app struct {
		Application struct {
			ID       string `json:"id"`
			ClientID string `json:"client_id"`
		} `json:"application"`
		ClientSecret string `json:"client_secret"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/applications", map[string]any{
		"name": "the server itself", "type": "web",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{s.root + "/oauth2/social/" + socialSlug + "/callback"},
		"scopes":        []string{"openid", "profile", "email"},
	}, &app)

	if app.ClientSecret == "" {
		t.Fatal("the application was registered without a secret to use")
	}

	body := map[string]any{
		"kind": "oidc", "slug": socialSlug, "name": "the server",
		"client_id": app.Application.ClientID, "client_secret": app.ClientSecret,
		"authorize_url": s.root + "/oauth2/authorize",
		"token_url":     s.root + "/oauth2/token",
		"userinfo_url":  s.root + "/oauth2/userinfo",
		"scopes":        []string{"openid", "profile", "email"},
	}
	maps.Copy(body, provider)

	super.must(http.StatusCreated, http.MethodPost, "/social-providers", body, nil)

	return f
}

// user creates the account the sign-in at the provider will use.
func (f *socialFixture) user(t *testing.T, verified bool) {
	t.Helper()

	var created struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": f.email, "password": f.password, "confirm_password": f.password,
		"first_name": "Sam", "last_name": "Rivers",
		"is_email_verified": verified, "is_active": true,
	}, &created)

	f.userID = created.User.ID
}

// signInThere walks the browser through the provider's own sign-in, the way a
// person would: to the provider, in with a password there, and back through
// the callback. It returns where the callback sent the browser afterwards.
func (f *socialFixture) signInThere(t *testing.T, b *browser, start string) *url.URL {
	t.Helper()

	return b.visit(f.atCallback(t, b, start))
}

// atCallback walks the browser as far as the provider's answer — the callback
// address, with a code — without going through it.
func (f *socialFixture) atCallback(t *testing.T, b *browser, start string) string {
	t.Helper()

	// To the provider — which is this server's authorization endpoint — and
	// on to its sign-in page, carrying the handle for the sign-in there.
	atProvider := b.visit(start)
	if !strings.HasPrefix(atProvider.String(), f.s.root+"/oauth2/authorize") {
		t.Fatalf("start led to %s, want the provider's authorization endpoint", atProvider)
	}
	if atProvider.Query().Get("code_challenge") == "" {
		t.Error("the sign-in was sent without PKCE")
	}

	signInPage := b.visit(atProvider.String())
	handle := signInPage.Query().Get("request")
	if handle == "" {
		t.Fatalf("the provider led to %s, want its sign-in page with a request", signInPage)
	}

	// The password sign-in at the provider, which answers with where to go
	// back to: our own callback, with a code.
	var result struct {
		RedirectTo string `json:"redirect_to"`
	}
	if status := b.account(http.MethodPost, "/login", map[string]any{
		"request": handle, "email": f.email, "password": f.password,
	}, &result); status != http.StatusOK {
		t.Fatalf("signing in at the provider = %d", status)
	}

	if !strings.Contains(result.RedirectTo, "/oauth2/social/"+f.slug+"/callback") {
		t.Fatalf("the provider sent the browser to %s, want the callback", result.RedirectTo)
	}

	return result.RedirectTo
}

// A provider's answer signs in the browser that started the sign-in and no
// other. The state goes to the provider and comes back in the address, so
// whoever starts a sign-in of their own could otherwise hand the finished
// address to somebody else's browser and leave it signed in as them — and
// everything that person did next would go into the attacker's account.
func TestLiveSocialCallbackOnlySignsInTheBrowserThatStarted(t *testing.T) {
	f := newSocialFixture(t, nil)
	f.user(t, true)

	starter := f.s.browser()
	callback := f.atCallback(t, starter, f.s.root+"/oauth2/social/"+f.slug+"/start")

	intruder := f.s.browser()
	if landed := intruder.visit(callback); landed.Query().Get("reason") != "social_expired" {
		t.Fatalf("another browser's callback landed on %s, want social_expired", landed)
	}
	if status := intruder.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("the other browser was signed in by a callback it never started")
	}

	// Nor did the attempt spend the sign-in: the browser that started it
	// still finishes.
	if landed := starter.visit(callback); landed.Query().Get("reason") != "" {
		t.Fatalf("the browser that started the sign-in landed on %s", landed)
	}
	if status := starter.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Errorf("GET /me = %d, want the browser that started the sign-in signed in", status)
	}
}

// A user who already has an account here signs in with the provider: the
// address the provider says is verified is enough to link the two, and the
// session that comes out is a session like any other.
func TestLiveSocialSignInLinksAVerifiedAddress(t *testing.T) {
	f := newSocialFixture(t, nil)
	f.user(t, true)

	b := f.s.browser()
	landed := f.signInThere(t, b, f.s.root+"/oauth2/social/"+f.slug+"/start")

	if strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the sign-in failed: %s", landed.Query().Get("error_description"))
	}
	if !strings.HasPrefix(landed.String(), testAccountURL) {
		t.Errorf("the callback landed on %s, want the account app", landed)
	}

	// The session the callback set is the user's, and it works.
	var me struct {
		User struct {
			Email string `json:"email"`
			ID    string `json:"id"`
		} `json:"user"`
	}
	if status := b.account(http.MethodGet, "/me", nil, &me); status != http.StatusOK {
		t.Fatalf("the session the callback set = %d, want a signed-in user", status)
	}
	if me.User.Email != f.email {
		t.Errorf("signed in as %q, want %q", me.User.Email, f.email)
	}
	if me.User.ID != f.userID {
		t.Errorf("a second account was made instead of using the one that had the address")
	}

	// Signing in again finds the identity rather than linking it twice, and
	// the log says what happened both times.
	again := f.s.browser()
	f.signInThere(t, again, f.s.root+"/oauth2/social/"+f.slug+"/start")

	var logs struct {
		Logs []struct {
			Action   string         `json:"action"`
			Metadata map[string]any `json:"metadata"`
		} `json:"logs"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/logs?limit=100", nil, &logs)

	connected := 0
	for _, entry := range logs.Logs {
		if entry.Action == "user.identity_connected" {
			connected++
		}
	}
	if connected != 1 {
		t.Errorf("the identity was connected %d times, want once", connected)
	}
	if identities := f.identities(t); identities != 1 {
		t.Errorf("%d identities are connected, want one", identities)
	}
}

// An address the provider will not vouch for is refused rather than linked:
// otherwise anybody who could make an account at the provider with somebody
// else's address could take over their account here.
func TestLiveSocialSignInRefusesAnUnverifiedAddress(t *testing.T) {
	f := newSocialFixture(t, nil)
	allowUnverifiedAddresses(t, f.super)
	f.user(t, false)

	b := f.s.browser()
	landed := f.signInThere(t, b, f.s.root+"/oauth2/social/"+f.slug+"/start")

	if !strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the callback landed on %s, want the error page", landed)
	}
	if got := landed.Query().Get("error_description"); !strings.Contains(got, "Sign in with your password") {
		t.Errorf("the page was told %q, want the sentence that says what to do", got)
	}
	if got := landed.Query().Get("reason"); got != "social_link_refused" {
		t.Errorf("reason = %q, want the code the page says in the reader's language", got)
	}

	if identities := f.identities(t); identities != 0 {
		t.Errorf("%d identities were connected, want none", identities)
	}
}

// identities is how many users hold an identity at the fixture's provider, as
// the panel is told.
func (f *socialFixture) identities(t *testing.T) int64 {
	t.Helper()

	var list struct {
		Providers []struct {
			Slug       string `json:"slug"`
			Identities int64  `json:"identities"`
		} `json:"providers"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/social-providers", nil, &list)

	for _, provider := range list.Providers {
		if provider.Slug == f.slug {
			return provider.Identities
		}
	}

	t.Fatalf("the provider %q is not in the list", f.slug)
	return 0
}

// The same, when the provider is one this installation does not trust to
// prove an address at all.
func TestLiveSocialSignInHonoursLinkingBeingOff(t *testing.T) {
	f := newSocialFixture(t, map[string]any{"link_verified_emails": false})
	f.user(t, true)

	landed := f.signInThere(t, f.s.browser(), f.s.root+"/oauth2/social/"+f.slug+"/start")

	if !strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Errorf("the callback landed on %s, want the error page", landed)
	}
}

// A provider that is turned off is not a way in, and neither is a slug
// nobody registered.
func TestLiveSocialDisabledProviderIsNotAWayIn(t *testing.T) {
	f := newSocialFixture(t, map[string]any{"is_enabled": false})

	b := f.s.browser()
	landed := b.visit(f.s.root + "/oauth2/social/" + f.slug + "/start")

	if !strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Errorf("a disabled provider led to %s, want the error page", landed)
	}

	unknown := b.visit(f.s.root + "/oauth2/social/nobody/start")
	if !strings.HasPrefix(unknown.String(), testAccountURL+"/error") {
		t.Errorf("an unknown provider led to %s, want the error page", unknown)
	}

	// The sign-in pages are told about neither.
	res, err := http.Get(f.s.root + "/api/v1/account/social-providers")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var buttons struct {
		Providers []struct {
			Slug string `json:"slug"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&buttons); err != nil {
		t.Fatal(err)
	}
	if len(buttons.Providers) != 0 {
		t.Errorf("the sign-in pages were offered %+v, want nothing", buttons.Providers)
	}
}

// An answer from a provider is only good once, and only for the sign-in it
// was started for.
func TestLiveSocialCallbackIsSingleUse(t *testing.T) {
	f := newSocialFixture(t, nil)
	f.user(t, true)

	b := f.s.browser()

	// Walk the flow by hand so the callback address can be used twice.
	atProvider := b.visit(f.s.root + "/oauth2/social/" + f.slug + "/start")
	signInPage := b.visit(atProvider.String())

	var result struct {
		RedirectTo string `json:"redirect_to"`
	}
	b.account(http.MethodPost, "/login", map[string]any{
		"request": signInPage.Query().Get("request"),
		"email":   f.email, "password": f.password,
	}, &result)

	if landed := b.visit(result.RedirectTo); strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the first time through failed: %s", landed.Query().Get("error_description"))
	}

	replayed := b.visit(result.RedirectTo)
	if !strings.HasPrefix(replayed.String(), testAccountURL+"/error") {
		t.Errorf("the same answer was accepted twice: %s", replayed)
	}

	// A made-up state is refused too.
	forged := b.visit(f.s.root + "/oauth2/social/" + f.slug + "/callback?code=whatever&state=made-up")
	if !strings.HasPrefix(forged.String(), testAccountURL+"/error") {
		t.Errorf("a made-up state was accepted: %s", forged)
	}
}

// The panel's own endpoints: what they answer with, and what they never do.
func TestLiveSocialProvidersAreManaged(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	type providerBody struct {
		Provider struct {
			ID              string   `json:"id"`
			Slug            string   `json:"slug"`
			Name            string   `json:"name"`
			Kind            string   `json:"kind"`
			Scopes          []string `json:"scopes"`
			HasClientSecret bool     `json:"has_client_secret"`
			CallbackURL     string   `json:"callback_url"`
			Enabled         bool     `json:"is_enabled"`
		} `json:"provider"`
	}

	var created providerBody
	super.must(http.StatusCreated, http.MethodPost, "/social-providers", map[string]any{
		"kind": "google", "name": "Google",
		"client_id": "1234.apps.googleusercontent.com", "client_secret": "a secret",
	}, &created)

	if created.Provider.Slug != "google" || created.Provider.Kind != "google" {
		t.Errorf("created = %+v", created.Provider)
	}
	if !created.Provider.HasClientSecret {
		t.Error("the panel was not told the secret is stored")
	}
	if want := s.root + "/oauth2/social/google/callback"; created.Provider.CallbackURL != want {
		t.Errorf("callback = %q, want %q", created.Provider.CallbackURL, want)
	}
	// The kind's own scopes, since none were asked for.
	if len(created.Provider.Scopes) == 0 {
		t.Error("a provider was created with nothing to ask the provider for")
	}

	// The secret never comes back, however it is asked for.
	var raw json.RawMessage
	super.must(http.StatusOK, http.MethodGet, "/social-providers/"+created.Provider.ID, nil, &raw)
	if strings.Contains(string(raw), "a secret") {
		t.Error("the endpoint answered with the secret it was given")
	}

	// An update that says nothing about the secret keeps it.
	var updated providerBody
	super.must(http.StatusOK, http.MethodPatch, "/social-providers/"+created.Provider.ID, map[string]any{
		"name": "Google Workspace", "client_id": "1234.apps.googleusercontent.com", "is_enabled": false,
	}, &updated)

	if !updated.Provider.HasClientSecret || updated.Provider.Name != "Google Workspace" || updated.Provider.Enabled {
		t.Errorf("updated = %+v", updated.Provider)
	}

	// Two providers cannot share an identifier, since it is in the address
	// each registers with its provider.
	super.must(http.StatusConflict, http.MethodPost, "/social-providers", map[string]any{
		"kind": "google", "slug": "google", "name": "Google again",
		"client_id": "another", "client_secret": "another secret",
	}, nil)

	// The list carries the kinds a new one may be, so the panel can offer
	// them without knowing any of this itself.
	var list struct {
		Providers []struct {
			Slug string `json:"slug"`
		} `json:"providers"`
		Kinds []struct {
			Kind   string `json:"kind"`
			Label  string `json:"label"`
			Custom bool   `json:"custom"`
		} `json:"kinds"`
	}
	super.must(http.StatusOK, http.MethodGet, "/social-providers", nil, &list)

	if len(list.Providers) != 1 {
		t.Errorf("%d providers, want one", len(list.Providers))
	}
	if len(list.Kinds) < 5 {
		t.Errorf("%d kinds offered, want every one this server knows", len(list.Kinds))
	}

	// The secret can be read back by whoever could replace it, and every
	// reading is in the log with who asked.
	var revealed struct {
		Secret string `json:"secret"`
		Kind   string `json:"kind"`
	}
	super.must(http.StatusOK, http.MethodGet, "/social-providers/"+created.Provider.ID+"/secret", nil, &revealed)

	if revealed.Secret != "a secret" || revealed.Kind != "client_secret" {
		t.Errorf("revealed = %+v, want the secret that was stored", revealed)
	}

	var logs struct {
		Logs []struct {
			Action   string         `json:"action"`
			Metadata map[string]any `json:"metadata"`
		} `json:"logs"`
	}
	super.must(http.StatusOK, http.MethodGet, "/logs?limit=50", nil, &logs)

	read := 0
	for _, entry := range logs.Logs {
		if entry.Action == "social_provider.secret_read" {
			read++
		}
	}
	if read != 1 {
		t.Errorf("the log has %d readings of the secret, want one", read)
	}

	super.must(http.StatusNoContent, http.MethodDelete, "/social-providers/"+created.Provider.ID, nil, nil)
	super.must(http.StatusNotFound, http.MethodGet, "/social-providers/"+created.Provider.ID, nil, nil)

	// An administrator whose roles say nothing about providers cannot see
	// them, let alone register one.
	manager := s.appManager(super, super.application("shop"))
	manager.must(http.StatusForbidden, http.MethodGet, "/social-providers", nil, nil)
	manager.must(http.StatusForbidden, http.MethodPost, "/social-providers", map[string]any{
		"kind": "google", "name": "Theirs", "client_id": "id", "client_secret": "s",
	}, nil)
}

// A user's record says which providers they sign in with, and an
// administrator can take one away without touching the account.
func TestLiveUserSocialAccountsAreShown(t *testing.T) {
	f := newSocialFixture(t, nil)
	f.user(t, true)

	f.signInThere(t, f.s.browser(), f.s.root+"/oauth2/social/"+f.slug+"/start")

	type userBody struct {
		User struct {
			Email          string `json:"email"`
			SocialAccounts []struct {
				ID       string `json:"id"`
				Provider string `json:"provider"`
				Slug     string `json:"slug"`
				Kind     string `json:"kind"`
				Email    string `json:"email"`
			} `json:"social_accounts"`
		} `json:"user"`
	}

	var got userBody
	f.super.must(http.StatusOK, http.MethodGet, "/users/"+f.userID, nil, &got)

	if len(got.User.SocialAccounts) != 1 {
		t.Fatalf("%d providers on the record, want one", len(got.User.SocialAccounts))
	}

	account := got.User.SocialAccounts[0]
	if account.Slug != f.slug || account.Kind != "oidc" || account.Email != f.email {
		t.Errorf("account = %+v", account)
	}

	// The list says so too, so the table can mark the rows.
	var page struct {
		Users []struct {
			Email          string `json:"email"`
			SocialAccounts []struct {
				Slug string `json:"slug"`
			} `json:"social_accounts"`
		} `json:"users"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/users?search="+f.email, nil, &page)

	if len(page.Users) != 1 || len(page.Users[0].SocialAccounts) != 1 {
		t.Errorf("the list says %+v, want the one provider", page.Users)
	}

	// Disconnected, the account stays and the mark goes.
	f.super.must(http.StatusNoContent, http.MethodDelete,
		"/users/"+f.userID+"/social-accounts/"+account.ID, nil, nil)

	f.super.must(http.StatusOK, http.MethodGet, "/users/"+f.userID, nil, &got)
	if len(got.User.SocialAccounts) != 0 {
		t.Errorf("the provider is still connected: %+v", got.User.SocialAccounts)
	}
	if got.User.Email != f.email {
		t.Errorf("the account itself changed: %+v", got.User)
	}

	// An identity that is not this user's is not theirs to disconnect.
	f.super.must(http.StatusNotFound, http.MethodDelete,
		"/users/"+f.userID+"/social-accounts/"+account.ID, nil, nil)
}

// A provider of our own making, for what the server above cannot show.
//
// Signing in through this server itself proves the flow against a real
// OpenID Connect provider, but every account that can sign in there is
// already an account here — so it can never show a first sign-in making one.
// This one answers whatever a test needs it to: an address nobody here has,
// one it will not vouch for, or none at all.
type fakeProvider struct {
	server *httptest.Server

	// claims is what its profile endpoint answers with.
	claims map[string]any
	// basic says the client has to authenticate with HTTP Basic.
	basic bool

	// seen is what the last token request carried, so a test can check what
	// this server sent.
	seenAuth     string
	seenVerifier string
	seenRedirect string
}

func newFakeProvider(t *testing.T, claims map[string]any) *fakeProvider {
	t.Helper()

	f := &fakeProvider{claims: claims}

	mux := http.NewServeMux()

	// Straight back with a code: a person is not being asked anything here.
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		back, err := url.Parse(r.URL.Query().Get("redirect_uri"))
		if err != nil {
			http.Error(w, "no redirect_uri", http.StatusBadRequest)
			return
		}

		query := back.Query()
		query.Set("code", "a-code")
		query.Set("state", r.URL.Query().Get("state"))
		back.RawQuery = query.Encode()

		http.Redirect(w, r, back.String(), http.StatusFound)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		f.seenAuth = r.Header.Get("Authorization")
		f.seenVerifier = r.PostFormValue("code_verifier")
		f.seenRedirect = r.PostFormValue("redirect_uri")

		if f.basic && f.seenAuth == "" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}
		if !f.basic && r.PostFormValue("client_secret") == "" {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "an-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer an-access-token" {
			http.Error(w, `{"error":"invalid_token"}`, http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.claims)
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)

	return f
}

// register configures the server to sign users in through this provider, and
// returns a browser and the address that starts a sign-in.
func (f *fakeProvider) register(t *testing.T, s *liveServer, super *client, extra map[string]any) (*browser, string) {
	t.Helper()

	body := map[string]any{
		"kind": "oidc", "slug": "fake", "name": "Example ID",
		"client_id": "fake-client", "client_secret": "a secret",
		"authorize_url": f.server.URL + "/authorize",
		"token_url":     f.server.URL + "/token",
		"userinfo_url":  f.server.URL + "/userinfo",
		"token_auth":    "post",
	}
	maps.Copy(body, extra)
	if body["token_auth"] == "basic" {
		f.basic = true
	}

	super.must(http.StatusCreated, http.MethodPost, "/social-providers", body, nil)

	return s.browser(), s.root + "/oauth2/social/fake/start"
}

// through walks the browser from the start address to wherever the callback
// leaves it.
func through(t *testing.T, b *browser, start string) *url.URL {
	t.Helper()

	atProvider := b.visit(start)
	back := b.visit(atProvider.String())

	return b.visit(back.String())
}

// A first sign-in with an address nobody here has makes the account, with the
// name the provider gave.
func TestLiveSocialSignInCreatesAnAccount(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	provider := newFakeProvider(t, map[string]any{
		"sub": "provider-subject-1", "email": "new@example.com", "email_verified": true,
		"given_name": "Ada", "family_name": "Lovelace",
	})

	b, start := provider.register(t, s, super, nil)
	landed := through(t, b, start)

	if strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the sign-in failed: %s", landed.Query().Get("error_description"))
	}

	// The account exists, with what the provider said about it.
	var users struct {
		Users []struct {
			Email         string `json:"email"`
			FirstName     string `json:"first_name"`
			LastName      string `json:"last_name"`
			EmailVerified bool   `json:"is_email_verified"`
		} `json:"users"`
	}
	super.must(http.StatusOK, http.MethodGet, "/users?search=new@example.com", nil, &users)

	if len(users.Users) != 1 {
		t.Fatalf("%d accounts were made, want one", len(users.Users))
	}
	if got := users.Users[0]; got.FirstName != "Ada" || got.LastName != "Lovelace" || !got.EmailVerified {
		t.Errorf("the account is %+v, want the name and the verified address the provider gave", got)
	}

	// And the browser is signed in as them.
	var me struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
	}
	if status := b.account(http.MethodGet, "/me", nil, &me); status != http.StatusOK {
		t.Fatalf("/me = %d, want the new account signed in", status)
	}
	if me.User.Email != "new@example.com" {
		t.Errorf("signed in as %q", me.User.Email)
	}

	// The code was bound to this sign-in, and spent against the address the
	// provider was told to send people back to.
	if provider.seenVerifier == "" {
		t.Error("the code was spent without its PKCE verifier")
	}
	if want := s.root + "/oauth2/social/fake/callback"; provider.seenRedirect != want {
		t.Errorf("redirect_uri = %q, want %q", provider.seenRedirect, want)
	}
}

// A provider that gives no address has nothing to make an account with, and
// says so rather than making a nameless one.
func TestLiveSocialSignInWithoutAnAddress(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	provider := newFakeProvider(t, map[string]any{"sub": "provider-subject-2"})
	b, start := provider.register(t, s, super, nil)

	landed := through(t, b, start)

	if !strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the sign-in went through without an address: %s", landed)
	}
	if got := landed.Query().Get("error_description"); !strings.Contains(got, "email address") {
		t.Errorf("the page was told %q, want what was missing", got)
	}
}

// A provider that may not make accounts turns away someone it has never seen,
// even with an address.
func TestLiveSocialSignInWithRegistrationOff(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	provider := newFakeProvider(t, map[string]any{
		"sub": "provider-subject-3", "email": "stranger@example.com", "email_verified": true,
	})
	b, start := provider.register(t, s, super, map[string]any{"allow_registration": false})

	landed := through(t, b, start)

	if !strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("an account was made anyway: %s", landed)
	}
	if got := landed.Query().Get("error_description"); !strings.Contains(got, "new accounts") {
		t.Errorf("the page was told %q", got)
	}
	if got := landed.Query().Get("reason"); got != "social_registration_closed" {
		t.Errorf("reason = %q", got)
	}
}

// The name is taken apart when a provider gives only one, and the address is
// matched whatever case it arrives in.
func TestLiveSocialSignInReadsWhatTheProviderGives(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	provider := newFakeProvider(t, map[string]any{
		"sub": "provider-subject-4", "email": "Grace.Hopper@Example.com", "email_verified": true,
		"name": "Grace Brewster Hopper",
	})
	b, start := provider.register(t, s, super, nil)

	if landed := through(t, b, start); strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the sign-in failed: %s", landed.Query().Get("error_description"))
	}

	var users struct {
		Users []struct {
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"users"`
	}
	super.must(http.StatusOK, http.MethodGet, "/users?search=hopper", nil, &users)

	if len(users.Users) != 1 {
		t.Fatalf("%d accounts, want one", len(users.Users))
	}

	got := users.Users[0]
	if got.Email != "grace.hopper@example.com" {
		t.Errorf("email = %q, want it stored lower case", got.Email)
	}
	if got.FirstName != "Grace" || got.LastName != "Brewster Hopper" {
		t.Errorf("name = %q %q, want the one name split in two", got.FirstName, got.LastName)
	}
}

// The secret goes the way the provider asked for it, and only that way.
func TestLiveSocialTokenAuthentication(t *testing.T) {
	for _, method := range []string{"basic", "post"} {
		t.Run(method, func(t *testing.T) {
			s := newLiveServer(t)
			super := s.superAdmin()

			provider := newFakeProvider(t, map[string]any{
				"sub": "subject-" + method, "email": method + "@example.com", "email_verified": true,
			})
			b, start := provider.register(t, s, super, map[string]any{"token_auth": method})

			if landed := through(t, b, start); strings.HasPrefix(landed.String(), testAccountURL+"/error") {
				t.Fatalf("the sign-in failed: %s", landed.Query().Get("error_description"))
			}

			if method == "basic" && provider.seenAuth == "" {
				t.Error("the secret was not sent as HTTP Basic")
			}
			if method == "post" && provider.seenAuth != "" {
				t.Error("the secret was sent as HTTP Basic as well as in the body")
			}
		})
	}
}
