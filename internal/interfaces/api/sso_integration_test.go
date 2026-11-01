package api

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"html"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/crewjam/saml"
	"github.com/crewjam/saml/logger"
	dsig "github.com/russellhaering/goxmldsig"

	"vard/internal/brand"
	"vard/internal/jose"
)

// Enterprise single sign-on, end to end, against identity providers that are
// real enough to be wrong in the ways real ones are: an OpenID Connect
// provider that signs its id_tokens with a key it publishes, and a SAML one —
// crewjam's own identity provider — that signs its assertions with the
// certificate in its metadata. Each says whoever the test tells it is signing
// in, so provisioning, domains and group-to-role mapping can all be seen.

// fakeOIDC is an OpenID Connect provider with one client.
type fakeOIDC struct {
	t      *testing.T
	server *httptest.Server
	key    jose.Key
	// forger signs with a key the provider does not publish, when set.
	forger *jose.Key
	// others are keys it publishes beside key, and never signs with.
	others []jose.Key

	mu     sync.Mutex
	person map[string]any
	codes  map[string]fakeCode
	// userinfo, when set, is what its userinfo endpoint answers, unsigned.
	userinfo map[string]any
}

type fakeCode struct {
	nonce, challenge, redirect string
}

const fakeClientID, fakeClientSecret = "fake-client", "fake-client-secret"

// fakeScopes are the scopes the fake provider knows.
var fakeScopes = []string{"openid", "email", "profile"}

func newFakeOIDC(t *testing.T) *fakeOIDC {
	t.Helper()

	private, err := jose.Generate(jose.RS256)
	if err != nil {
		t.Fatal(err)
	}

	f := &fakeOIDC{t: t, key: jose.Key{ID: "fake-1", Algorithm: jose.RS256, Private: private}, codes: map[string]fakeCode{}}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		withUserinfo := f.userinfo != nil
		f.mu.Unlock()
		document := map[string]any{
			"issuer":                                f.server.URL,
			"authorization_endpoint":                f.server.URL + "/authorize",
			"token_endpoint":                        f.server.URL + "/token",
			"jwks_uri":                              f.server.URL + "/jwks",
			"token_endpoint_auth_methods_supported": []string{"client_secret_basic"},
			"scopes_supported":                      fakeScopes,
		}
		if withUserinfo {
			document["userinfo_endpoint"] = f.server.URL + "/userinfo"
		}
		_ = json.NewEncoder(w).Encode(document)
	})
	mux.HandleFunc("GET /userinfo", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(f.userinfo)
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		var published []any
		for _, key := range append([]jose.Key{f.key}, f.others...) {
			public, _ := key.Public()
			published = append(published, public)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": published})
	})
	mux.HandleFunc("GET /authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("client_id") != fakeClientID || q.Get("code_challenge_method") != "S256" || q.Get("nonce") == "" {
			http.Error(w, "bad authorization request", http.StatusBadRequest)
			return
		}

		back, _ := url.Parse(q.Get("redirect_uri"))
		values := back.Query()
		values.Set("state", q.Get("state"))

		// A scope it does not know refuses the whole sign-in, the way
		// Keycloak does.
		for scope := range strings.FieldsSeq(q.Get("scope")) {
			if !slices.Contains(fakeScopes, scope) {
				values.Set("error", "invalid_scope")
				values.Set("error_description", "Invalid scopes: "+q.Get("scope"))
				back.RawQuery = values.Encode()
				http.Redirect(w, r, back.String(), http.StatusFound)
				return
			}
		}

		code := randomHex(t, 8)
		f.mu.Lock()
		f.codes[code] = fakeCode{nonce: q.Get("nonce"), challenge: q.Get("code_challenge"), redirect: q.Get("redirect_uri")}
		f.mu.Unlock()

		values.Set("code", code)
		back.RawQuery = values.Encode()

		http.Redirect(w, r, back.String(), http.StatusFound)
	})
	mux.HandleFunc("POST /token", func(w http.ResponseWriter, r *http.Request) {
		id, secret, ok := r.BasicAuth()
		if !ok || id != fakeClientID || secret != fakeClientSecret {
			http.Error(w, `{"error":"invalid_client"}`, http.StatusUnauthorized)
			return
		}

		f.mu.Lock()
		code, known := f.codes[r.PostFormValue("code")]
		delete(f.codes, r.PostFormValue("code"))
		person := f.person
		f.mu.Unlock()

		sum := sha256.Sum256([]byte(r.PostFormValue("code_verifier")))
		if !known || base64.RawURLEncoding.EncodeToString(sum[:]) != code.challenge || r.PostFormValue("redirect_uri") != code.redirect {
			http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
			return
		}

		claims := map[string]any{
			"iss": f.server.URL, "aud": fakeClientID, "nonce": code.nonce,
			"iat": time.Now().Unix(), "exp": time.Now().Add(5 * time.Minute).Unix(),
		}
		maps.Copy(claims, person)

		signer := f.key
		if f.forger != nil {
			signer = *f.forger
		}
		token, err := jose.Sign(signer, "JWT", claims)
		if err != nil {
			t.Error(err)
		}

		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": token})
	})

	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)

	return f
}

// signs says who the provider signs in next.
func (f *fakeOIDC) signs(claims map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.person = claims
}

// ssoResult is where a sign-in through a connection ended up.
type ssoResult struct {
	landed *url.URL
}

func (r ssoResult) failed() (string, bool) {
	if strings.HasPrefix(r.landed.String(), testAccountURL+"/error") {
		return r.landed.Query().Get("reason"), true
	}
	return "", false
}

// signInOIDC walks a browser through a connection: to the provider, which
// signs whoever it was told, and back.
func (f *fakeOIDC) signInOIDC(b *browser, slug string) ssoResult {
	f.t.Helper()

	atProvider := b.visit(b.s.root + "/oauth2/sso/" + slug + "/start?next=/&login_hint=someone%40acme.test")
	if !strings.HasPrefix(atProvider.String(), f.server.URL+"/authorize") {
		f.t.Fatalf("start led to %s, want the provider", atProvider)
	}
	if atProvider.Query().Get("login_hint") != "someone@acme.test" {
		f.t.Error("the address typed was not passed on as login_hint")
	}

	callback := b.visit(atProvider.String())

	return ssoResult{landed: b.visit(callback.String())}
}

// me is who a browser is signed in as.
func (b *browser) me() (email, first, id string) {
	b.t.Helper()

	var out struct {
		User struct {
			ID        string `json:"id"`
			Email     string `json:"email"`
			FirstName string `json:"first_name"`
		} `json:"user"`
	}
	if status := b.account(http.MethodGet, "/me", nil, &out); status != http.StatusOK {
		b.t.Fatalf("GET /me = %d, want the browser signed in", status)
	}

	return out.User.Email, out.User.FirstName, out.User.ID
}

// roleNames are the roles an administrator sees a user holding directly.
func (c *client) roleNames(userID string) []string {
	var out struct {
		User struct {
			Roles []struct {
				Name string `json:"name"`
			} `json:"roles"`
		} `json:"user"`
	}
	c.must(http.StatusOK, http.MethodGet, "/users/"+userID, nil, &out)

	names := make([]string, 0, len(out.User.Roles))
	for _, role := range out.User.Roles {
		names = append(names, role.Name)
	}

	return names
}

func TestLiveSSOWithOpenIDConnect(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)
	engineer := super.role("engineer", "")

	var created struct {
		Connection struct {
			ID              string `json:"id"`
			HasClientSecret bool   `json:"has_client_secret"`
			ServiceProvider struct {
				CallbackURL string `json:"callback_url"`
			} `json:"service_provider"`
		} `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"ACME.test", "@acme.test"}, "is_enabled": true,
		"role_mappings": []map[string]string{{"group": "Engineering", "role_id": engineer}},
		"sync_roles":    true,
	}, &created)

	if !created.Connection.HasClientSecret {
		t.Error("the secret was not kept")
	}
	if want := s.root + "/oauth2/sso/acme/callback"; created.Connection.ServiceProvider.CallbackURL != want {
		t.Errorf("callback = %q, want %q", created.Connection.ServiceProvider.CallbackURL, want)
	}

	// Somebody new: an account is made, named by the provider, verified, and
	// given the role their group maps to.
	idp.signs(map[string]any{
		"sub": "okta|ada", "email": "ada@acme.test", "email_verified": true,
		"given_name": "Ada", "family_name": "Lovelace", "groups": []string{"engineering", "staff"},
	})

	b := s.browser()
	result := idp.signInOIDC(b, "acme")
	if reason, failed := result.failed(); failed {
		t.Fatalf("the first sign-in failed: %s", reason)
	}

	email, first, userID := b.me()
	if email != "ada@acme.test" || first != "Ada" {
		t.Errorf("signed in as %s (%s), want the provider's person", email, first)
	}
	if roles := super.roleNames(userID); len(roles) != 1 || roles[0] != "engineer" {
		t.Errorf("roles = %v, want the one the group maps to", roles)
	}

	// Again, out of the group and renamed at the provider: the same account,
	// the new name, and the mapped role taken away.
	idp.signs(map[string]any{
		"sub": "okta|ada", "email": "ada@acme.test", "given_name": "Augusta", "groups": []string{"staff"},
	})
	again := s.browser()
	if reason, failed := idp.signInOIDC(again, "acme").failed(); failed {
		t.Fatalf("the second sign-in failed: %s", reason)
	}
	if _, first, id := again.me(); id != userID || first != "Augusta" {
		t.Errorf("second sign-in = %s %s, want the same account renamed", id, first)
	}
	if roles := super.roleNames(userID); len(roles) != 0 {
		t.Errorf("roles = %v, want the mapped role gone with the group", roles)
	}

	tests := []struct {
		name   string
		person map[string]any
		forge  bool
		want   string
	}{
		{
			name:   "an address outside the connection's domains",
			person: map[string]any{"sub": "okta|eve", "email": "eve@evil.test"},
			want:   "sso_domain_mismatch",
		},
		{
			name:   "an address the provider says it has not verified",
			person: map[string]any{"sub": "okta|bob", "email": "bob@acme.test", "email_verified": false},
			want:   "sso_no_email",
		},
		{
			name:   "an address the provider says, as a string, it has not verified",
			person: map[string]any{"sub": "okta|bea", "email": "bea@acme.test", "email_verified": "false"},
			want:   "sso_no_email",
		},
		{
			name:   "an id_token signed with a key the provider does not publish",
			person: map[string]any{"sub": "okta|ada", "email": "ada@acme.test"},
			forge:  true,
			want:   "sso_upstream",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idp.signs(tt.person)
			idp.forger = nil
			if tt.forge {
				forged, _ := jose.Generate(jose.RS256)
				idp.forger = &jose.Key{ID: "fake-1", Algorithm: jose.RS256, Private: forged}
			}
			defer func() { idp.forger = nil }()

			reason, failed := idp.signInOIDC(s.browser(), "acme").failed()
			if !failed || reason != tt.want {
				t.Errorf("the sign-in ended with %q (failed %v), want %s", reason, failed, tt.want)
			}
		})
	}

	// An answer is only accepted once: the callback, replayed, is refused.
	idp.signs(map[string]any{"sub": "okta|ada", "email": "ada@acme.test"})
	replayer := s.browser()
	atProvider := replayer.visit(s.root + "/oauth2/sso/acme/start")
	callback := replayer.visit(atProvider.String())
	replayer.visit(callback.String())
	if reason, failed := (ssoResult{landed: replayer.visit(callback.String())}).failed(); !failed || reason != "sso_expired" {
		t.Errorf("a replayed callback ended with %q, want sso_expired", reason)
	}

	// Enforced, the domain's password ways in point at the connection.
	super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, map[string]any{
		"enforce_domains": true,
	}, nil)

	var refused problemBody
	if status := s.browser().account(http.MethodPost, "/login", map[string]string{
		"email": "ada@acme.test", "password": "whatever-it-is",
	}, &refused); status != http.StatusForbidden || refused.Code != "sso_required" || refused.Params["slug"] != "acme" {
		t.Errorf("a password sign-in at an enforced domain = %d %+v, want sso_required naming the connection", status, refused)
	}

	var found struct {
		Connection struct{ Slug, Name string } `json:"connection"`
		Enforced   bool                        `json:"enforced"`
	}
	if status := s.browser().account(http.MethodPost, "/sso/discover", map[string]string{"email": "grace@Acme.test"}, &found); status != http.StatusOK ||
		found.Connection.Slug != "acme" || !found.Enforced {
		t.Errorf("discovering an address at the domain = %d %+v", status, found)
	}
	if status := s.browser().account(http.MethodPost, "/sso/discover", map[string]string{"email": "grace@elsewhere.test"}, &refused); status != http.StatusNotFound || refused.Code != "no_sso_connection" {
		t.Errorf("discovering an address elsewhere = %d %+v, want no_sso_connection", status, refused)
	}

	// A domain belongs to one connection.
	var taken problemBody
	super.do(http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme again", "issuer": idp.server.URL, "client_id": "x",
		"domains": []string{"acme.test"},
	}, &taken)
	if taken.Code != "sso_domain_taken" || taken.Params["domain"] != "acme.test" {
		t.Errorf("a second connection for the domain = %+v, want sso_domain_taken", taken)
	}
}

// The connection's matching and provisioning settings decide what happens to
// somebody who already has an account, or none.
func TestLiveSSOMatchingAndProvisioning(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	var grace struct {
		User idOnly `json:"user"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "grace@acme.test", "first_name": "Grace", "password": "grace-password-1", "confirm_password": "grace-password-1",
	}, &grace)

	var created struct {
		Connection idOnly `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"acme.test"}, "is_enabled": true, "matching": "deny", "create_users": false,
	}, &created)

	idp.signs(map[string]any{"sub": "okta|grace", "email": "grace@acme.test"})
	if reason, _ := idp.signInOIDC(s.browser(), "acme").failed(); reason != "sso_link_refused" {
		t.Errorf("an existing account at a connection that refuses to link = %q, want sso_link_refused", reason)
	}

	idp.signs(map[string]any{"sub": "okta|new", "email": "new@acme.test"})
	if reason, _ := idp.signInOIDC(s.browser(), "acme").failed(); reason != "sso_no_account" {
		t.Errorf("somebody new at a connection that makes no accounts = %q, want sso_no_account", reason)
	}

	super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, map[string]any{"matching": "link"}, nil)

	// Whoever registered an address before its owner arrived through the
	// provider would keep the password they set, so an account that never
	// verified its address is not linked.
	idp.signs(map[string]any{"sub": "okta|grace", "email": "grace@acme.test"})
	if reason, _ := idp.signInOIDC(s.browser(), "acme").failed(); reason != "sso_link_refused" {
		t.Errorf("linking an account that never verified its address = %q, want sso_link_refused", reason)
	}

	super.must(http.StatusOK, http.MethodPatch, "/users/"+grace.User.ID, map[string]any{
		"email": "grace@acme.test", "first_name": "Grace", "is_email_verified": true,
	}, nil)

	// Nor at a domain the connection has not proved it holds.
	if reason, _ := idp.signInOIDC(s.browser(), "acme").failed(); reason != "sso_link_refused" {
		t.Errorf("linking at a domain not proved = %q, want sso_link_refused", reason)
	}
	s.provedDomain(t, created.Connection.ID, "acme.test")

	b := s.browser()
	if reason, failed := idp.signInOIDC(b, "acme").failed(); failed {
		t.Fatalf("linking an existing account failed: %s", reason)
	}
	if email, first, _ := b.me(); email != "grace@acme.test" || first != "Grace" {
		t.Errorf("linked to %s (%s), want Grace's own account, unrenamed", email, first)
	}

	// Off, a connection signs nobody in.
	super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, map[string]any{"is_enabled": false}, nil)
	off := s.browser().visit(s.root + "/oauth2/sso/acme/start")
	if off.Query().Get("reason") != "sso_unknown" {
		t.Errorf("a connection that is off sent the browser to %s", off)
	}
}

// fakeSAML is a SAML identity provider: crewjam's own, signing in whoever it
// is told.
type fakeSAML struct {
	server *httptest.Server
	idp    *saml.IdentityProvider
	mu     sync.Mutex
	person *saml.Session
}

func newFakeSAML(t *testing.T) *fakeSAML {
	t.Helper()

	key, err := jose.Generate(jose.RS256)
	if err != nil {
		t.Fatal(err)
	}
	certificatePEM, err := jose.SelfSigned(key, "fake-idp", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	certificate, _ := jose.ParseCertificate(certificatePEM)

	f := &fakeSAML{}
	mux := http.NewServeMux()
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)

	base, _ := url.Parse(f.server.URL)
	f.idp = &saml.IdentityProvider{
		Key:                     key,
		Signer:                  key,
		Logger:                  logger.DefaultLogger,
		Certificate:             certificate,
		MetadataURL:             *base.JoinPath("metadata"),
		SSOURL:                  *base.JoinPath("sso"),
		ServiceProviderProvider: f,
		SessionProvider:         f,
		// crewjam's own default is SHA-1, which the ACS refuses.
		SignatureMethod: dsig.RSASHA256SignatureMethod,
	}

	mux.HandleFunc("/sso", f.idp.ServeSSO)

	return f
}

func (f *fakeSAML) GetSession(w http.ResponseWriter, _ *http.Request, _ *saml.IdpAuthnRequest) *saml.Session {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.person
}

// GetServiceProvider reads our metadata the way a real provider is set up
// from it: from its address.
func (f *fakeSAML) GetServiceProvider(_ *http.Request, id string) (*saml.EntityDescriptor, error) {
	res, err := http.Get(id)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var descriptor saml.EntityDescriptor
	body, _ := io.ReadAll(res.Body)
	return &descriptor, xml.Unmarshal(body, &descriptor)
}

func (f *fakeSAML) metadata(t *testing.T) string {
	t.Helper()

	document, err := xml.Marshal(f.idp.Metadata())
	if err != nil {
		t.Fatal(err)
	}

	return string(document)
}

var formField = regexp.MustCompile(`name="(SAMLResponse|RelayState)" value="([^"]*)"`)

// signInSAML walks a browser through a SAML connection: to the provider with
// a request, back with the form it posts. It returns the form too, for a test
// that wants to post it again.
func (f *fakeSAML) signInSAML(t *testing.T, b *browser, slug string) (ssoResult, url.Values) {
	t.Helper()

	atProvider := b.visit(b.s.root + "/oauth2/sso/" + slug + "/start")
	if !strings.HasPrefix(atProvider.String(), f.server.URL+"/sso") || atProvider.Query().Get("SAMLRequest") == "" {
		t.Fatalf("start led to %s, want the provider with a request", atProvider)
	}

	res, err := http.Get(atProvider.String())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	form := url.Values{}
	for _, match := range formField.FindAllStringSubmatch(string(body), -1) {
		form.Set(match[1], html.UnescapeString(match[2]))
	}
	if form.Get("SAMLResponse") == "" {
		t.Fatalf("the provider answered %d without a response: %.300s", res.StatusCode, body)
	}

	return b.post(t, b.s.root+"/oauth2/sso/"+slug+"/acs", form), form
}

// providerForm walks a browser to the provider and returns the form the
// provider's page would post back, without posting it.
func (f *fakeSAML) providerForm(t *testing.T, b *browser, slug string) url.Values {
	t.Helper()

	atProvider := b.visit(b.s.root + "/oauth2/sso/" + slug + "/start")
	res, err := http.Get(atProvider.String())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	form := url.Values{}
	for _, match := range formField.FindAllStringSubmatch(string(body), -1) {
		form.Set(match[1], html.UnescapeString(match[2]))
	}
	if form.Get("SAMLResponse") == "" || form.Get("RelayState") == "" {
		t.Fatalf("the provider answered %d without a response: %.300s", res.StatusCode, body)
	}

	return form
}

// post sends a form as the provider's page would, and returns where it led,
// following a 303 back to this server as a browser does (the SAML ACS holds
// the post and finishes on the GET).
func (b *browser) post(t *testing.T, target string, form url.Values) ssoResult {
	t.Helper()

	res, err := b.http.PostForm(target, form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	landed, err := url.Parse(res.Header.Get("Location"))
	if err != nil || (res.StatusCode != http.StatusFound && res.StatusCode != http.StatusSeeOther) {
		t.Fatalf("POST %s = %d, want a redirect", target, res.StatusCode)
	}
	if res.StatusCode == http.StatusSeeOther {
		landed = b.visit(landed.String())
	}

	return ssoResult{landed: landed}
}

// postAcross sends a form as another site's page would over plain http: the
// browser's SameSite=Lax cookies stay behind. It returns where it led, for
// the browser itself to follow.
func postAcross(t *testing.T, target string, form url.Values) *url.URL {
	t.Helper()

	noCookies := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := noCookies.PostForm(target, form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	landed, err := url.Parse(res.Header.Get("Location"))
	if err != nil || (res.StatusCode != http.StatusFound && res.StatusCode != http.StatusSeeOther) {
		t.Fatalf("POST %s = %d, want a redirect", target, res.StatusCode)
	}

	return landed
}

func TestLiveSSOWithSAML(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeSAML(t)
	auditor := super.role("auditors", "")

	var created struct {
		Connection struct {
			ID              string `json:"id"`
			ServiceProvider struct {
				ACSURL      string `json:"acs_url"`
				EntityID    string `json:"entity_id"`
				Certificate string `json:"certificate"`
			} `json:"service_provider"`
			IdentityProvider struct {
				EntityID string `json:"entity_id"`
				SSOURL   string `json:"sso_url"`
			} `json:"identity_provider"`
		} `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "saml", "name": "Globex", "metadata": idp.metadata(t),
		"domains": []string{"globex.test"}, "is_enabled": true, "name_id_format": "persistent",
		"groups_attribute": "eduPersonAffiliation",
		"role_mappings":    []map[string]string{{"group": "audit", "role_id": auditor}},
	}, &created)

	sp := created.Connection.ServiceProvider
	if sp.ACSURL != s.root+"/oauth2/sso/globex/acs" || sp.EntityID != s.root+"/oauth2/sso/globex/metadata" {
		t.Errorf("service provider = %+v", sp)
	}
	if _, err := jose.ParseCertificate(sp.Certificate); err != nil {
		t.Errorf("the connection was given no usable certificate: %v", err)
	}
	if created.Connection.IdentityProvider.SSOURL != idp.server.URL+"/sso" {
		t.Errorf("identity provider = %+v, want what its metadata says", created.Connection.IdentityProvider)
	}

	// The metadata our provider is set up from.
	res, err := http.Get(sp.EntityID)
	if err != nil {
		t.Fatal(err)
	}
	var ours saml.EntityDescriptor
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if err := xml.Unmarshal(body, &ours); err != nil || ours.EntityID != sp.EntityID ||
		ours.SPSSODescriptors[0].AssertionConsumerServices[0].Location != sp.ACSURL {
		t.Errorf("our metadata = %.300s (%v)", body, err)
	}

	idp.mu.Lock()
	idp.person = &saml.Session{
		ID: "session-1", CreateTime: time.Now(), ExpireTime: time.Now().Add(time.Hour),
		NameID: "globex-7", UserEmail: "hedy@globex.test", UserGivenName: "Hedy", UserSurname: "Lamarr",
		Groups: []string{"audit"},
	}
	idp.mu.Unlock()

	b := s.browser()
	result, form := idp.signInSAML(t, b, "globex")
	if reason, failed := result.failed(); failed {
		t.Fatalf("the SAML sign-in failed: %s", reason)
	}

	email, first, userID := b.me()
	if email != "hedy@globex.test" || first != "Hedy" {
		t.Errorf("signed in as %s (%s), want the assertion's person", email, first)
	}
	if roles := super.roleNames(userID); len(roles) != 1 || roles[0] != "auditors" {
		t.Errorf("roles = %v, want the one the group maps to", roles)
	}

	// The same response, posted again, is refused: it answered a request that
	// has already been answered.
	if reason, _ := s.browser().post(t, sp.ACSURL, form).failed(); reason != "sso_expired" {
		t.Errorf("a replayed response ended with %q, want sso_expired", reason)
	}

	// A response nobody asked for — IdP-initiated — is refused too.
	unsolicited := url.Values{"SAMLResponse": {form.Get("SAMLResponse")}}
	if reason, _ := s.browser().post(t, sp.ACSURL, unsolicited).failed(); reason != "sso_expired" {
		t.Errorf("an unsolicited response ended with %q, want sso_expired", reason)
	}

	// A response signed by anybody but the provider in the metadata is
	// refused: here, the metadata is swapped for another provider's.
	other := newFakeSAML(t)
	super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, map[string]any{
		"metadata": other.metadata(t),
	}, nil)
	// The request goes to the new provider's address; the answer comes from
	// the old one, which still has the session.
	intruder := s.browser()
	atOther := intruder.visit(s.root + "/oauth2/sso/globex/start")
	relay := atOther.Query().Get("RelayState")
	if reason, _ := intruder.post(t, sp.ACSURL, url.Values{"SAMLResponse": {form.Get("SAMLResponse")}, "RelayState": {relay}}).failed(); reason != "sso_upstream" {
		t.Errorf("a response from a provider not in the metadata ended with %q, want sso_upstream", reason)
	}
}

// samlConnection makes a SAML connection to a fake provider that signs Hedy
// in, and returns the provider.
func samlConnection(t *testing.T, s *liveServer, super *client, slug string) *fakeSAML {
	t.Helper()

	idp := newFakeSAML(t)
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "saml", "name": slug, "slug": slug, "metadata": idp.metadata(t),
		"domains": []string{slug + ".test"}, "is_enabled": true,
	}, nil)

	idp.mu.Lock()
	idp.person = &saml.Session{
		ID: "session-" + slug, CreateTime: time.Now(), ExpireTime: time.Now().Add(time.Hour),
		NameID: slug + "-7", UserEmail: "hedy@" + slug + ".test", UserGivenName: "Hedy",
	}
	idp.mu.Unlock()

	return idp
}

// Over plain http the provider's cross-site POST to the ACS carries none of
// the browser's cookies (SameSite=Lax). The ACS holds it and sends the browser
// back with a GET, which does carry the state cookie, and the sign-in
// finishes there — for the browser that started it, and no other.
func TestLiveSAMLWithoutCrossSiteCookies(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := samlConnection(t, s, super, "initech")
	acs := s.root + "/oauth2/sso/initech/acs"

	t.Run("the browser that started signs in", func(t *testing.T) {
		b := s.browser()
		form := idp.providerForm(t, b, "initech")

		back := postAcross(t, acs, form)
		if back.String() != acs {
			t.Fatalf("the ACS sent the browser to %s, want back to itself", back)
		}
		if reason, failed := (ssoResult{landed: b.visit(back.String())}).failed(); failed {
			t.Fatalf("the sign-in failed: %s", reason)
		}
		if email, _, _ := b.me(); email != "hedy@initech.test" {
			t.Errorf("signed in as %q", email)
		}
	})

	t.Run("RelayState is not the browser's state", func(t *testing.T) {
		b := s.browser()
		form := idp.providerForm(t, b, "initech")
		var state string
		for _, cookie := range b.http.Jar.Cookies(&url.URL{Scheme: "http", Host: strings.TrimPrefix(s.root, "http://"), Path: "/oauth2/"}) {
			if cookie.Name == brand.SignInStateCookie {
				state = cookie.Value
			}
		}
		if state == "" || form.Get("RelayState") == state {
			t.Errorf("RelayState %q is the state cookie %q", form.Get("RelayState"), state)
		}
		if len(form.Get("RelayState")) > 80 {
			t.Errorf("RelayState is %d bytes; the binding allows 80", len(form.Get("RelayState")))
		}
	})

	t.Run("another browser cannot finish it", func(t *testing.T) {
		starter := s.browser()
		form := idp.providerForm(t, starter, "initech")

		// The attacker's own sign-in, posted from the victim's browser.
		victim := s.browser()
		back := postAcross(t, acs, form)
		if reason, failed := (ssoResult{landed: victim.visit(back.String())}).failed(); !failed || reason != "sso_expired" {
			t.Errorf("another browser finished a sign-in it never started: %q", reason)
		}
		if status := victim.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
			t.Fatal("the other browser was signed in")
		}

		// Nor did the attempt spend it.
		if reason, failed := (ssoResult{landed: starter.visit(acs)}).failed(); failed {
			t.Errorf("the browser that started failed after another tried: %s", reason)
		}
	})

	t.Run("coming back early spends nothing", func(t *testing.T) {
		b := s.browser()
		form := idp.providerForm(t, b, "initech")

		// Anyone can link a browser to the ACS before its provider answers.
		if reason, _ := (ssoResult{landed: b.visit(acs)}).failed(); reason != "sso_expired" {
			t.Errorf("a GET before the provider answered ended with %q", reason)
		}

		back := postAcross(t, acs, form)
		if reason, failed := (ssoResult{landed: b.visit(back.String())}).failed(); failed {
			t.Errorf("the sign-in failed after an early GET: %s", reason)
		}
	})

	t.Run("one answer per sign-in", func(t *testing.T) {
		b := s.browser()
		form := idp.providerForm(t, b, "initech")
		postAcross(t, acs, form)

		if again := postAcross(t, acs, form); again.Query().Get("reason") != "sso_expired" {
			t.Errorf("a second answer to the same sign-in led to %s", again)
		}
		if reason, failed := (ssoResult{landed: b.visit(acs)}).failed(); failed {
			t.Errorf("the first answer was lost: %s", reason)
		}
	})
}

// What the ACS refuses that the SAML library would let through, each logged
// against the connection with its reason.
func TestLiveSAMLRefusesWhatTheLibraryWouldTake(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := samlConnection(t, s, super, "hooli")
	acs := s.root + "/oauth2/sso/hooli/acs"

	reasons := func() string {
		var logs struct {
			Logs []struct {
				Action string `json:"action"`
				Detail string `json:"detail"`
			} `json:"logs"`
		}
		super.must(http.StatusOK, http.MethodGet, "/logs?limit=5", nil, &logs)
		for _, entry := range logs.Logs {
			if entry.Action == "sso_connection.sign_in_failed" {
				return entry.Detail
			}
		}
		return ""
	}

	for _, tt := range []struct {
		name, reason string
		form         func(form url.Values)
		before       func()
	}{
		{
			name:   "an artifact instead of a response",
			reason: "only the HTTP-POST binding",
			form: func(form url.Values) {
				form.Del("SAMLResponse")
				form.Set("SAMLart", "AAQAAMh48/1oXIM+sDo7Dh2qMp1HM4IF5DaRNmDj6RdUmllwn9jJHyEgIi8=")
			},
		},
		{
			name:   "a SHA-1 signature",
			reason: "SHA-1",
			before: func() {
				idp.idp.SignatureMethod = dsig.RSASHA1SignatureMethod
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			idp.idp.SignatureMethod = dsig.RSASHA256SignatureMethod
			if tt.before != nil {
				tt.before()
			}

			b := s.browser()
			form := idp.providerForm(t, b, "hooli")
			if tt.form != nil {
				tt.form(form)
			}

			result := b.post(t, acs, form)
			if reason, failed := result.failed(); !failed || reason != "sso_upstream" {
				t.Errorf("ended with %q, want sso_upstream", reason)
			}
			if got := reasons(); !strings.Contains(got, tt.reason) {
				t.Errorf("logged %q, want it to say %q", got, tt.reason)
			}
		})
	}
}

// Metadata that is not a provider's is refused with a reason, before it is
// relied on.
// An identity provider's answer signs in the browser that started the sign-in
// and no other. The state goes to the provider and comes back in the address,
// so whoever starts a sign-in of their own could otherwise hand the finished
// address to somebody else's browser and leave it signed in as them.
func TestLiveSSOCallbackOnlySignsInTheBrowserThatStarted(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"acme.test"}, "is_enabled": true,
	}, nil)

	idp.signs(map[string]any{
		"sub": "okta|grace", "email": "grace@acme.test", "email_verified": true, "given_name": "Grace",
	})

	// As far as the provider's answer, without going through the callback.
	starter := s.browser()
	atProvider := starter.visit(s.root + "/oauth2/sso/acme/start")
	callback := starter.visit(atProvider.String())
	if !strings.HasPrefix(callback.String(), s.root+"/oauth2/sso/acme/callback") {
		t.Fatalf("the provider sent the browser to %s, want the callback", callback)
	}

	intruder := s.browser()
	if reason, failed := (ssoResult{landed: intruder.visit(callback.String())}).failed(); !failed || reason != "sso_expired" {
		t.Fatalf("another browser's callback ended with %q, want sso_expired", reason)
	}
	if status := intruder.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("the other browser was signed in by a callback it never started")
	}

	// Nor did the attempt spend the sign-in.
	if reason, failed := (ssoResult{landed: starter.visit(callback.String())}).failed(); failed {
		t.Fatalf("the browser that started the sign-in failed with %q", reason)
	}
	if email, _, _ := starter.me(); email != "grace@acme.test" {
		t.Errorf("signed in as %q, want grace@acme.test", email)
	}
}

func TestLiveSSORefusesBadMetadata(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var refused problemBody
	super.do(http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "saml", "name": "Broken", "metadata": "<not-metadata/>", "domains": []string{"broken.test"},
	}, &refused)
	if refused.Code != "sso_metadata_invalid" || refused.Params["reason"] == "" {
		t.Errorf("bad metadata = %+v, want sso_metadata_invalid with a reason", refused)
	}

	super.do(http.MethodPost, "/sso-connections/test", map[string]any{
		"protocol": "oidc", "issuer": "https://nowhere.invalid",
	}, &refused)
	if refused.Code != "sso_discovery_failed" {
		t.Errorf("an issuer nobody answers for = %+v, want sso_discovery_failed", refused)
	}
}

// A connection without domains is trusted with any address, as a source is in
// authentik — and since no address leads to it, only its button does.
func TestLiveSSOWithoutDomains(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	connection := map[string]any{
		"protocol": "oidc", "name": "Partners", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret, "is_enabled": true,
	}

	for _, tt := range []struct {
		name   string
		change map[string]any
	}{
		{name: "without its button, nothing leads to it", change: map[string]any{"show_on_login": false}},
		{name: "required of nobody", change: map[string]any{"show_on_login": true, "enforce_domains": true}},
	} {
		body := maps.Clone(connection)
		maps.Copy(body, tt.change)

		var refused problemBody
		if status := super.do(http.MethodPost, "/sso-connections", body, &refused); status != http.StatusBadRequest || refused.Code != "sso_unreachable" {
			t.Errorf("%s: = %d %+v, want sso_unreachable", tt.name, status, refused)
		}
	}

	connection["show_on_login"] = true
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", connection, nil)

	idp.signs(map[string]any{"sub": "partner|1", "email": "lin@anywhere.test", "given_name": "Lin"})
	b := s.browser()
	if reason, failed := idp.signInOIDC(b, "partners").failed(); failed {
		t.Fatalf("signing in through a connection without domains failed: %s", reason)
	}
	if email, _, _ := b.me(); email != "lin@anywhere.test" {
		t.Errorf("signed in as %q, want lin@anywhere.test", email)
	}

	// Its button is on the sign-in page, but "Sign in with SSO" is not: no
	// address would lead anywhere.
	var buttons struct {
		Connections []struct{ Slug string } `json:"connections"`
		Available   bool                    `json:"available"`
	}
	s.browser().account(http.MethodGet, "/sso", nil, &buttons)
	if len(buttons.Connections) != 1 || buttons.Connections[0].Slug != "partners" || buttons.Available {
		t.Errorf("the sign-in page's SSO = %+v, want the button and no discovery", buttons)
	}
}

// kid is optional in a JWS. A provider that publishes several keys and names
// none of them still signs people in, by whichever of its keys verifies; a key
// it does not publish, or a kid naming another of its keys, still does not.
func TestLiveSSOWithoutKeyIDs(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	key := func(id string) jose.Key {
		private, err := jose.Generate(jose.RS256)
		if err != nil {
			t.Fatal(err)
		}
		return jose.Key{ID: id, Algorithm: jose.RS256, Private: private}
	}
	idp.others = []jose.Key{key(""), key("")}
	idp.key.ID = ""

	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Keyless", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"keyless.test"}, "show_on_login": true, "is_enabled": true,
	}, nil)

	named := idp.key
	named.ID = "someone-else"

	for _, tt := range []struct {
		name   string
		forger *jose.Key
		signs  bool
	}{
		{name: "signed by one of several keys, none named", signs: true},
		{name: "signed by a key it does not publish", forger: new(key(""))},
		{name: "a kid naming a key it does not publish", forger: &named},
	} {
		idp.forger = tt.forger
		idp.signs(map[string]any{"sub": "keyless|1", "email": "kim@keyless.test", "email_verified": true})

		b := s.browser()
		reason, failed := idp.signInOIDC(b, "keyless").failed()
		if failed == tt.signs {
			t.Errorf("%s: signed in = %v (%s), want %v", tt.name, !failed, reason, tt.signs)
		}
	}
}

// A connection without domains may sign in any address but one another
// connection holds: a partner's provider asserting somebody at Acme must not
// become them, past Acme's own connection, whether that is on or off.
func TestLiveSSOWithoutDomainsLeavesOtherConnectionsDomainsAlone(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	acme, partner := newFakeOIDC(t), newFakeOIDC(t)

	var owner struct {
		Connection idOnly `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": acme.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"acme.test"}, "enforce_domains": true, "is_enabled": true,
	}, &owner)
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Partners", "issuer": partner.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"show_on_login": true, "is_enabled": true, "create_users": true,
	}, nil)
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "ceo@acme.test", "first_name": "Ceo", "password": "ceo-password-1",
		"confirm_password": "ceo-password-1", "is_email_verified": true,
	}, nil)

	tests := []struct {
		name    string
		enabled bool
		email   string
	}{
		{"an existing account at a domain another connection enforces", true, "ceo@acme.test"},
		{"a new address there", true, "new@acme.test"},
		{"the same, with the owner turned off", false, "ceo@acme.test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+owner.Connection.ID, map[string]any{"is_enabled": tt.enabled}, nil)
			partner.signs(map[string]any{"sub": "partner|" + tt.email, "email": tt.email, "email_verified": true})

			b := s.browser()
			if reason, failed := partner.signInOIDC(b, "partners").failed(); !failed || reason != "sso_domain_mismatch" {
				t.Errorf("the sign-in ended with %q (failed %v), want sso_domain_mismatch", reason, failed)
			}
		})
	}

	partner.signs(map[string]any{"sub": "partner|lin", "email": "lin@anywhere.test", "email_verified": true})
	if reason, failed := partner.signInOIDC(s.browser(), "partners").failed(); failed {
		t.Errorf("an address nobody holds was refused: %s", reason)
	}
}

// What a provider said when it did not complete a sign-in goes to the
// activity log against the connection, for whoever sets it up; the person
// signing in is only told it failed.
func TestLiveSSOProviderFailuresAreRecorded(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	// Test connection warns about a scope the provider does not list.
	var tested struct {
		UnsupportedScopes []string `json:"unsupported_scopes"`
	}
	super.must(http.StatusOK, http.MethodPost, "/sso-connections/test", map[string]any{
		"protocol": "oidc", "issuer": idp.server.URL, "scopes": []string{"email", "profile", "groups"},
	}, &tested)
	if !slices.Equal(tested.UnsupportedScopes, []string{"groups"}) {
		t.Errorf("unsupported scopes = %v, want [groups]", tested.UnsupportedScopes)
	}

	var created struct {
		Connection idOnly `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"acme.test"}, "is_enabled": true, "scopes": []string{"email", "profile", "groups"},
	}, &created)

	failures := func() []string {
		var logs struct {
			Logs []struct {
				Action string `json:"action"`
				Detail string `json:"detail"`
			} `json:"logs"`
		}
		super.must(http.StatusOK, http.MethodGet, "/logs?limit=100", nil, &logs)

		var out []string
		for _, entry := range logs.Logs {
			if entry.Action == "sso_connection.sign_in_failed" {
				out = append(out, entry.Detail)
			}
		}
		return out
	}

	ada := map[string]any{"sub": "okta|ada", "email": "ada@acme.test"}

	for _, tt := range []struct {
		name   string
		change map[string]any
		person map[string]any
		shown  string
		want   string
	}{
		{
			name:  "a scope the provider refuses",
			shown: "sso_upstream",
			want:  "authorize: invalid_scope: Invalid scopes: openid email profile groups",
		},
		{
			name:   "a client secret the provider does not know",
			change: map[string]any{"scopes": []string{}, "client_secret": "wrong"},
			shown:  "sso_upstream",
			want:   "token: ",
		},
		{
			name:   "an address the provider has not verified",
			change: map[string]any{"client_secret": fakeClientSecret},
			person: map[string]any{"sub": "okta|ada", "email": "ada@acme.test", "email_verified": false},
			shown:  "sso_no_email",
			want:   "claims: the provider says it has not verified ada@acme.test",
		},
		{
			// What was sent is named, so a missing mapper shows as one.
			name:   "no address at all",
			person: map[string]any{"sub": "okta|ada", "preferred_username": "ada"},
			shown:  "sso_no_email",
			want:   "claims: no address in email; the provider sent aud, exp, iat, iss, nonce, preferred_username, sub",
		},
		{
			name:   "an address outside the connection's domains",
			person: map[string]any{"sub": "okta|ada", "email": "ada@elsewhere.test"},
			shown:  "sso_domain_mismatch",
			want:   "claims: ada@elsewhere.test is outside the connection's domains",
		},
	} {
		if tt.change != nil {
			super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, tt.change, nil)
		}
		person := ada
		if tt.person != nil {
			person = tt.person
		}
		idp.signs(person)

		before := len(failures())
		if reason, _ := idp.signInOIDC(s.browser(), "acme").failed(); reason != tt.shown {
			t.Errorf("%s: the person was shown %q, want %s", tt.name, reason, tt.shown)
		}

		got := failures()
		if len(got) != before+1 || !strings.HasPrefix(got[0], tt.want) {
			t.Errorf("%s: the activity log says %q, want one more entry starting %q", tt.name, got, tt.want)
		}
	}

	// A refusal for a sign-in this server never started is somebody's link,
	// not the provider's answer, and is not written down.
	before := len(failures())
	landed := s.browser().visit(s.root + "/oauth2/sso/acme/callback?error=access_denied&state=made-up")
	if landed.Query().Get("reason") != "sso_expired" {
		t.Errorf("a refusal with a state nobody issued landed on %s, want sso_expired", landed)
	}
	if after := len(failures()); after != before {
		t.Errorf("a made-up refusal was written to the activity log")
	}
}

// A subject is only unique within the provider that issued it. A connection
// pointed at another provider forgets the identities it held, so the new
// provider's "1" is not taken for the old one's.
func TestLiveSSOForgetsIdentitiesOfAnotherProvider(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	before, after := newFakeOIDC(t), newFakeOIDC(t)

	var created struct {
		Connection idOnly `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": before.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret,
		"domains": []string{"acme.test"}, "is_enabled": true,
	}, &created)

	for _, person := range []map[string]any{
		{"sub": "1", "email": "ada@acme.test"},
		{"sub": "2", "email": "grace@acme.test"},
	} {
		before.signs(person)
		if reason, failed := before.signInOIDC(s.browser(), "acme").failed(); failed {
			t.Fatalf("signing %s in failed: %s", person["email"], reason)
		}
	}

	super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+created.Connection.ID, map[string]any{
		"issuer": after.server.URL,
	}, nil)
	s.provedDomain(t, created.Connection.ID, "acme.test")

	after.signs(map[string]any{"sub": "1", "email": "grace@acme.test"})
	b := s.browser()
	if reason, failed := after.signInOIDC(b, "acme").failed(); failed {
		t.Fatalf("signing in at the new provider failed: %s", reason)
	}
	if email, _, _ := b.me(); email != "grace@acme.test" {
		t.Errorf("the new provider's subject 1 signed in as %s, want grace@acme.test", email)
	}
}
