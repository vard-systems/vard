package model

import (
	"strings"
	"testing"
	"time"
)

// app returns an application of the given type with everything a valid one
// of that type needs, for a test to break one thing at a time.
func app(kind ApplicationType) Application {
	a := Application{
		Name:                 "Shop",
		Type:                 kind,
		GrantTypes:           []string{GrantAuthorizationCode, GrantRefreshToken},
		RedirectURIs:         []string{"https://shop.example.com/callback"},
		Scopes:               []string{ScopeOpenID, ScopeEmail},
		AccessTokenLifetime:  DefaultAccessTokenLifetime,
		IDTokenLifetime:      DefaultIDTokenLifetime,
		RefreshTokenLifetime: DefaultRefreshTokenLifetime,
	}

	if kind == AppM2M {
		a.GrantTypes = []string{GrantClientCredentials}
	}

	return a
}

func TestApplicationValidate(t *testing.T) {
	tests := []struct {
		name   string
		kind   ApplicationType
		change func(*Application)
		want   string // part of the message, or "" when the settings are fine
	}{
		{name: "a web app", kind: AppWeb, change: func(*Application) {}},
		{name: "a single-page app", kind: AppSPA, change: func(*Application) {}},
		{name: "a native app", kind: AppNative, change: func(*Application) {}},
		{name: "a machine-to-machine app", kind: AppM2M, change: func(*Application) {}},
		{
			name:   "a loopback redirect over http",
			kind:   AppNative,
			change: func(a *Application) { a.RedirectURIs = []string{"http://127.0.0.1:8123/callback"} },
		},
		{
			name:   "a native private-use scheme",
			kind:   AppNative,
			change: func(a *Application) { a.RedirectURIs = []string{"com.example.shop:/callback"} },
		},
		{
			name:   "a back-channel logout URI over https with a query",
			kind:   AppWeb,
			change: func(a *Application) { a.BackchannelLogoutURI = "https://shop.example.com/logout?from=idp" },
		},
		{
			name:   "a back-channel logout URI on loopback over http",
			kind:   AppWeb,
			change: func(a *Application) { a.BackchannelLogoutURI = "http://localhost:3000/logout" },
		},
		{
			name:   "a back-channel logout URI over plain http",
			kind:   AppWeb,
			change: func(a *Application) { a.BackchannelLogoutURI = "http://shop.example.com/logout" },
			want:   "backchannel_logout_uri",
		},
		{
			name:   "a back-channel logout URI with a fragment",
			kind:   AppWeb,
			change: func(a *Application) { a.BackchannelLogoutURI = "https://shop.example.com/logout#now" },
			want:   "must not have a fragment",
		},
		{
			name:   "a relative back-channel logout URI",
			kind:   AppWeb,
			change: func(a *Application) { a.BackchannelLogoutURI = "/logout" },
			want:   "not an absolute URL",
		},
		{
			name:   "a front-channel logout URI with a fragment",
			kind:   AppSPA,
			change: func(a *Application) { a.FrontchannelLogoutURI = "https://shop.example.com/logout#now" },
			want:   "frontchannel_logout_uri",
		},
		{
			name: "a television app that signs in with the device code alone",
			kind: AppNative,
			change: func(a *Application) {
				a.GrantTypes = []string{GrantDeviceCode, GrantRefreshToken}
				a.RedirectURIs = nil
			},
		},
		{
			name:   "an unknown type",
			kind:   "desktop",
			change: func(*Application) {},
			want:   "type must be one of",
		},
		{
			name:   "the implicit grant",
			kind:   AppSPA,
			change: func(a *Application) { a.GrantTypes = []string{GrantAuthorizationCode, "implicit"} },
			want:   `"implicit" is not offered`,
		},
		{
			name:   "the password grant",
			kind:   AppWeb,
			change: func(a *Application) { a.GrantTypes = []string{"password"} },
			want:   `"password" is not offered`,
		},
		{
			name:   "a web app without authorization_code",
			kind:   AppWeb,
			change: func(a *Application) { a.GrantTypes = []string{GrantClientCredentials} },
			want:   "needs authorization_code",
		},
		{
			name:   "a public client asking for client_credentials",
			kind:   AppSPA,
			change: func(a *Application) { a.GrantTypes = []string{GrantAuthorizationCode, GrantClientCredentials} },
			want:   "cannot use client_credentials",
		},
		{
			name:   "no redirect URI",
			kind:   AppWeb,
			change: func(a *Application) { a.RedirectURIs = nil },
			want:   "at least one redirect URI",
		},
		{
			name:   "a wildcard redirect",
			kind:   AppWeb,
			change: func(a *Application) { a.RedirectURIs = []string{"https://*.example.com/callback"} },
			want:   "wildcards are not allowed",
		},
		{
			name:   "plain http off localhost",
			kind:   AppWeb,
			change: func(a *Application) { a.RedirectURIs = []string{"http://shop.example.com/callback"} },
			want:   "must use https",
		},
		{
			name:   "a fragment",
			kind:   AppWeb,
			change: func(a *Application) { a.RedirectURIs = []string{"https://shop.example.com/callback#done"} },
			want:   "must not have a fragment",
		},
		{
			name:   "a script URL",
			kind:   AppNative,
			change: func(a *Application) { a.RedirectURIs = []string{"javascript:alert(1)"} },
			want:   "must use https",
		},
		{
			name:   "a private-use scheme on a web app",
			kind:   AppWeb,
			change: func(a *Application) { a.RedirectURIs = []string{"com.example.shop:/callback"} },
			want:   "must use https",
		},
		{
			name:   "a logo over http",
			kind:   AppWeb,
			change: func(a *Application) { a.LogoURL = "http://shop.example.com/logo.png" },
			want:   "logo_url must be an https URL",
		},
		{
			name:   "a home page on localhost over http",
			kind:   AppWeb,
			change: func(a *Application) { a.WebsiteURL = "http://localhost:3000" },
		},
		{
			name:   "a home page over http anywhere",
			kind:   AppWeb,
			change: func(a *Application) { a.WebsiteURL = "http://shop.example.com" },
		},
		{
			name:   "a home page that is not a web address",
			kind:   AppWeb,
			change: func(a *Application) { a.WebsiteURL = "javascript:alert(1)" },
			want:   "website_url must be an http or https URL",
		},
		{
			name:   "a home page with no host",
			kind:   AppWeb,
			change: func(a *Application) { a.WebsiteURL = "http://" },
			want:   "website_url must be an http or https URL",
		},
		{
			name:   "an access token that outlives a day",
			kind:   AppWeb,
			change: func(a *Application) { a.AccessTokenLifetime = 2 * 24 * 60 * 60 },
			want:   "access_token_lifetime must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := app(tt.kind)
			tt.change(&a)
			a.Normalize()

			err := a.Validate()

			if tt.want == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nothing", err)
				}
				return
			}

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() = %v, want it to say %q", err, tt.want)
			}
		})
	}
}

// Normalize is where a type's rules are applied rather than refused: a public
// client can never end up with a secret or without PKCE, and a service has
// nothing to redirect to.
func TestApplicationNormalize(t *testing.T) {
	spa := app(AppSPA)
	spa.TokenAuthMethod = AuthClientSecretPost
	spa.RequirePKCE = false
	spa.Normalize()

	if spa.TokenAuthMethod != AuthNone || !spa.RequirePKCE || spa.HasSecret() {
		t.Errorf("spa: auth = %s, pkce = %v, want none and true", spa.TokenAuthMethod, spa.RequirePKCE)
	}

	web := app(AppWeb)
	web.TokenAuthMethod = AuthNone
	web.GrantTypes = []string{GrantRefreshToken, GrantAuthorizationCode, GrantRefreshToken}
	web.RedirectURIs = []string{" https://a.example.com/cb ", "", "https://a.example.com/cb"}
	web.Normalize()

	if web.TokenAuthMethod != AuthClientSecretBasic {
		t.Errorf("web: auth = %s, want client_secret_basic", web.TokenAuthMethod)
	}
	if got := strings.Join(web.GrantTypes, ","); got != "authorization_code,refresh_token" {
		t.Errorf("web: grant types = %s, want them once each in catalog order", got)
	}
	if len(web.RedirectURIs) != 1 || web.RedirectURIs[0] != "https://a.example.com/cb" {
		t.Errorf("web: redirect URIs = %q, want one tidy URI", web.RedirectURIs)
	}
	if got := web.ResponseTypes(); len(got) != 1 || got[0] != "code" {
		t.Errorf("web: response types = %v, want [code]", got)
	}

	m2m := app(AppM2M)
	m2m.GrantTypes = []string{GrantAuthorizationCode}
	m2m.RedirectURIs = []string{"https://a.example.com/cb"}
	m2m.BackchannelLogoutURI = "https://a.example.com/logout"
	m2m.Normalize()

	if m2m.BackchannelLogoutURI != "" {
		t.Errorf("m2m: back-channel logout URI = %q, want none: a service signs nobody in", m2m.BackchannelLogoutURI)
	}

	if len(m2m.GrantTypes) != 1 || m2m.GrantTypes[0] != GrantClientCredentials || len(m2m.RedirectURIs) != 0 {
		t.Errorf("m2m: grants = %v, redirects = %v, want client_credentials only and none", m2m.GrantTypes, m2m.RedirectURIs)
	}
	if len(m2m.ResponseTypes()) != 0 {
		t.Errorf("m2m: response types = %v, want none", m2m.ResponseTypes())
	}
}

func TestApplicationSecret(t *testing.T) {
	web := app(AppWeb)
	web.Normalize()

	secret, err := web.IssueSecret(time.Now())
	if err != nil {
		t.Fatalf("IssueSecret() = %v", err)
	}

	if len(secret) < 40 || web.ClientSecretHash == "" || web.ClientSecretHash == secret {
		t.Fatalf("secret %q, hash %q: want a long secret stored only as a hash", secret, web.ClientSecretHash)
	}
	if web.SecretHint != secret[len(secret)-4:] || web.SecretCreatedAt == nil {
		t.Errorf("hint = %q, created = %v", web.SecretHint, web.SecretCreatedAt)
	}
	if !web.CheckSecret(secret) || web.CheckSecret(secret+"x") || web.CheckSecret("") {
		t.Error("CheckSecret accepts the wrong secret or refuses the right one")
	}

	rotated, _ := web.IssueSecret(time.Now())
	if web.CheckSecret(secret) || !web.CheckSecret(rotated) {
		t.Error("the old secret still works after rotating")
	}

	spa := app(AppSPA)
	spa.Normalize()
	if _, err := spa.IssueSecret(time.Now()); err != ErrNoSecret {
		t.Errorf("IssueSecret() on a public client = %v, want ErrNoSecret", err)
	}
}

func TestNewClientID(t *testing.T) {
	a, _ := NewClientID()
	b, _ := NewClientID()

	if len(a) != 32 || a == b {
		t.Errorf("client ids %q and %q: want 32 hex characters, different each time", a, b)
	}
}

// A client with no credentials of its own has nothing but PKCE to tie a code
// to the app that asked for it, so it always uses PKCE, whatever its
// RequirePKCE flag says (RFC 9700 §2.1.1).
func TestApplicationMustUsePKCE(t *testing.T) {
	tests := []struct {
		name   string
		method AuthMethod
		flag   bool
		want   bool
	}{
		{"a public client with the flag off", AuthNone, false, true},
		{"a public client with the flag on", AuthNone, true, true},
		{"a confidential client with the flag off", AuthClientSecretBasic, false, false},
		{"a confidential client with the flag on", AuthClientSecretBasic, true, true},
		{"a mutual TLS client with the flag off", AuthTLSClient, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Application{TokenAuthMethod: tt.method, RequirePKCE: tt.flag}
			if got := a.MustUsePKCE(); got != tt.want {
				t.Errorf("MustUsePKCE() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Redirect URIs match exactly, except that a native app's loopback one
// matches on any port, since the app opens whichever is free (RFC 8252
// §7.3). Nothing else about it may differ, and no other kind of app gets it.
func TestApplicationAllowsRedirect(t *testing.T) {
	registered := []string{
		"http://127.0.0.1/callback",
		"http://[::1]:8080/callback",
		"com.example.shop:/callback",
		"https://shop.example.com/callback",
	}

	tests := []struct {
		name string
		kind ApplicationType
		uri  string
		want bool
	}{
		{"exactly a registered one", AppNative, "https://shop.example.com/callback", true},
		{"a private-use scheme, exactly", AppNative, "com.example.shop:/callback", true},
		{"loopback on the port it was registered without", AppNative, "http://127.0.0.1/callback", true},
		{"loopback on another port", AppNative, "http://127.0.0.1:51234/callback", true},
		{"IPv6 loopback on another port", AppNative, "http://[::1]:61000/callback", true},
		{"loopback with another path", AppNative, "http://127.0.0.1:51234/other", false},
		{"loopback with a query added", AppNative, "http://127.0.0.1:51234/callback?next=x", false},
		{"loopback with a user in it", AppNative, "http://evil@127.0.0.1:51234/callback", false},
		{"another loopback address than registered", AppNative, "http://[::1]:51234/other", false},
		{"https on another port", AppNative, "https://shop.example.com:8443/callback", false},
		{"a host that is not loopback", AppNative, "http://127.0.0.1.evil.test:51234/callback", false},
		{"a web app's loopback on another port", AppWeb, "http://127.0.0.1:51234/callback", false},
		{"a single-page app's loopback on another port", AppSPA, "http://127.0.0.1:51234/callback", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Application{Type: tt.kind, RedirectURIs: registered}
			if got := a.AllowsRedirect(tt.uri); got != tt.want {
				t.Errorf("AllowsRedirect(%q) = %v, want %v", tt.uri, got, tt.want)
			}
		})
	}
}

// Expo Go is how a native app is developed, and it answers on exp://. Those
// addresses name a developer's machine, so only a native app marked as in
// development may register one, and no other kind of application at all.
func TestApplicationExpoRedirect(t *testing.T) {
	tests := []struct {
		name        string
		kind        ApplicationType
		development bool
		uri         string
		want        string
	}{
		{"a native app in development, in Expo Go", AppNative, true, "exp://192.168.1.20:8081/--/callback", ""},
		{"a native app in development, in Expo Go over https", AppNative, true, "exps://u.expo.dev/abc/--/callback", ""},
		{"a released native app", AppNative, false, "exp://192.168.1.20:8081/--/callback", "for development"},
		{"a released native app, as a post-logout redirect", AppNative, false, "", "for development"},
		{"a single-page app", AppSPA, true, "exp://192.168.1.20:8081/--/callback", "must use https"},
		{"a web app", AppWeb, true, "exp://192.168.1.20:8081/--/callback", "must use https"},
		{"a native app with a scheme that is no domain", AppNative, true, "myapp://callback", "must use https"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := app(tt.kind)
			a.AllowDevelopmentRedirects = tt.development
			a.RedirectURIs = []string{tt.uri}
			if tt.uri == "" {
				a.PostLogoutRedirectURIs = []string{"exp://192.168.1.20:8081/--/"}
				a.RedirectURIs = []string{"com.example.shop:/callback"}
			}
			a.Normalize()

			err := a.Validate()
			if tt.want == "" && err != nil {
				t.Fatalf("Validate() = %v, want nothing", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("Validate() = %v, want it to say %q", err, tt.want)
			}
		})
	}
}
