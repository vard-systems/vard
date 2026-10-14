package model

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fixture is the example the design was agreed on: an orders API, a Shop web
// application allowed to read and write orders, and a support role granting
// read and refund.
type fixture struct {
	app                     Application
	orders                  API
	read, write, refund     APIScope
	support, shopAdmin, emp UserRole
	user                    User
}

func newFixture() fixture {
	f := fixture{}

	f.app = Application{
		Name:                "Shop web",
		Type:                AppWeb,
		ClientID:            "shop-web",
		GrantTypes:          []string{GrantAuthorizationCode, GrantRefreshToken},
		Scopes:              []string{ScopeOpenID, ScopeProfile, ScopeEmail, ScopeOfflineAccess, ScopeRoles},
		AccessTokenLifetime: 3600,
		IDTokenLifetime:     3600,
		AssertRoles:         true,
		IsEnabled:           true,
	}
	f.app.ID = uuid.New()

	scope := func(name string) APIScope {
		s := APIScope{Name: name,
			ID: uuid.New()}
		return s
	}
	f.read, f.write, f.refund = scope("orders:read"), scope("orders:write"), scope("orders:refund")

	f.orders = API{
		Name:               "Orders",
		Identifier:         "https://api.shop.com/orders",
		EnforceRoles:       true,
		SigningAlgorithm:   AlgRS256,
		AllowOfflineAccess: true,
		Scopes:             []APIScope{f.read, f.write, f.refund},
	}
	f.orders.ID = uuid.New()

	f.support = UserRole{Name: "support", APIScopes: []APIScope{f.read, f.refund}}
	f.support.ID = uuid.New()

	f.shopAdmin = UserRole{Name: "admin", ApplicationID: &f.app.ID}
	f.shopAdmin.ID = uuid.New()

	f.emp = UserRole{Name: "employee"}
	f.emp.ID = uuid.New()

	f.user = User{Email: "mira@example.com", FirstName: "Mira", LastName: "Testova", IsEmailVerified: true, IsActive: true}
	f.user.ID = uuid.New()

	return f
}

// request asks for a token for the user, for the orders API, with scopes.
func (f fixture) request(scope string) TokenRequest {
	return TokenRequest{
		Application: f.app,
		User:        &f.user,
		Roles:       []UserRole{f.support, f.shopAdmin, f.emp},
		API:         &f.orders,
		Authorized:  true,
		Allowed:     []string{"orders:read", "orders:write"},
		Requested:   strings.Fields(scope),
		Issuer:      "https://auth.example.com",
		Now:         time.Unix(1_800_000_000, 0),
	}
}

func decision(p TokenPreview, scope string) ScopeDecision {
	for _, d := range p.Decisions {
		if d.Scope == scope {
			return d
		}
	}
	return ScopeDecision{}
}

// The agreed example: support asks for read and refund through Shop web, and
// gets read only — refund is not something the application may carry.
func TestEvaluateTokenDownScopes(t *testing.T) {
	f := newFixture()

	p := EvaluateToken(f.request("orders:read orders:refund"))

	if !p.Issued {
		t.Fatalf("no token: %s", p.Reason)
	}
	if got := p.AccessToken["scope"]; got != "orders:read" {
		t.Errorf("scope = %q, want orders:read", got)
	}
	if got := p.AccessToken["aud"]; !reflect.DeepEqual(got, []string{"https://api.shop.com/orders"}) {
		t.Errorf("aud = %v, want the orders API alone", got)
	}
	if d := decision(p, "orders:refund"); d.Granted || d.Reason != "the application is not allowed this scope" {
		t.Errorf("refund = %+v, want refused as not allowed", d)
	}
	if d := decision(p, "orders:read"); !d.Granted || d.Reason != "granted by support" {
		t.Errorf("read = %+v, want granted by support", d)
	}
}

func TestEvaluateTokenScopeRules(t *testing.T) {
	tests := []struct {
		name    string
		change  func(*fixture, *TokenRequest)
		scope   string
		granted bool
		reason  string
	}{
		{
			name:    "allowed, but no role grants it",
			scope:   "orders:write",
			granted: false,
			reason:  "none of the user's roles grant this scope",
		},
		{
			name:    "allowed, and the API does not enforce roles",
			change:  func(f *fixture, r *TokenRequest) { r.API.EnforceRoles = false },
			scope:   "orders:write",
			granted: true,
			reason:  "the application is allowed it, and the API does not enforce roles",
		},
		{
			name:    "a scope the API does not have",
			scope:   "orders:delete",
			granted: false,
			reason:  "https://api.shop.com/orders has no such scope",
		},
		{
			name:    "an API scope with no audience",
			change:  func(f *fixture, r *TokenRequest) { r.API = nil },
			scope:   "orders:read",
			granted: false,
			reason:  "no audience was requested, so API scopes cannot be granted",
		},
		{
			name:    "an OpenID scope the application may use",
			scope:   "email",
			granted: true,
			reason:  "the application may use this scope",
		},
		{
			name:    "an OpenID scope the application may not use",
			change:  func(f *fixture, r *TokenRequest) { r.Application.Scopes = []string{ScopeOpenID} },
			scope:   "email",
			granted: false,
			reason:  "the application may not use this scope",
		},
		{
			name:    "offline access without the refresh grant",
			change:  func(f *fixture, r *TokenRequest) { r.Application.GrantTypes = []string{GrantAuthorizationCode} },
			scope:   "offline_access",
			granted: false,
			reason:  "a refresh token needs the refresh_token grant",
		},
		{
			name: "a scope granted through a composite role's inheritance",
			change: func(f *fixture, r *TokenRequest) {
				// Roles are already resolved: an inherited role is simply
				// among them.
				editor := UserRole{Name: "editor", APIScopes: []APIScope{f.write},
					ID: uuid.New()}
				r.Roles = append(r.Roles, editor)
			},
			scope:   "orders:write",
			granted: true,
			reason:  "granted by editor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			r := f.request(tt.scope)
			if tt.change != nil {
				tt.change(&f, &r)
			}

			p := EvaluateToken(r)
			if !p.Issued {
				t.Fatalf("no token: %s", p.Reason)
			}

			d := decision(p, tt.scope)
			if d.Granted != tt.granted || d.Reason != tt.reason {
				t.Errorf("decision = %+v, want granted=%v %q", d, tt.granted, tt.reason)
			}
		})
	}
}

func TestEvaluateTokenRefusals(t *testing.T) {
	tests := []struct {
		name   string
		change func(*fixture, *TokenRequest)
		reason string
	}{
		{
			name:   "a disabled application",
			change: func(f *fixture, r *TokenRequest) { r.Application.IsEnabled = false },
			reason: "the application is disabled",
		},
		{
			name:   "an application not authorised for the API",
			change: func(f *fixture, r *TokenRequest) { r.Authorized = false },
			reason: "the application is not authorised for https://api.shop.com/orders",
		},
		{
			name: "an inactive user",
			change: func(f *fixture, r *TokenRequest) {
				r.User.IsActive = false
			},
			reason: "the user is inactive",
		},
		{
			name: "a required role the user does not hold",
			change: func(f *fixture, r *TokenRequest) {
				r.Application.RequireRoleAssignment = true
				r.Roles = []UserRole{f.support, f.emp}
			},
			reason: "the application requires a role, and the user holds none of its roles",
		},
		{
			name:   "no user, and no client_credentials grant",
			change: func(f *fixture, r *TokenRequest) { r.User = nil },
			reason: "a token without a user needs the client_credentials grant, which the application does not have",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture()
			r := f.request("openid orders:read")
			tt.change(&f, &r)

			p := EvaluateToken(r)
			if p.Issued || p.Reason != tt.reason {
				t.Errorf("issued = %v, reason = %q; want refused: %q", p.Issued, p.Reason, tt.reason)
			}
		})
	}
}

func TestEvaluateTokenClaims(t *testing.T) {
	f := newFixture()

	p := EvaluateToken(f.request("openid profile email roles orders:read"))

	if p.IDToken["aud"] != "shop-web" || p.IDToken["email"] != "mira@example.com" || p.IDToken["name"] != "Mira Testova" {
		t.Errorf("id token = %v", p.IDToken)
	}
	if got := p.AccessToken[ClaimApplicationRoles]; !reflect.DeepEqual(got, []string{"admin"}) {
		t.Errorf("roles = %v, want this application's roles only", got)
	}
	if got := p.AccessToken[ClaimGlobalRoles]; !reflect.DeepEqual(got, []string{"employee", "support"}) {
		t.Errorf("global_roles = %v", got)
	}
	if p.AccessToken["exp"].(int64)-p.AccessToken["iat"].(int64) != 3600 {
		t.Errorf("lifetime = %v", p.AccessToken)
	}

	// Without "roles" in the request, no roles travel.
	quiet := EvaluateToken(f.request("openid orders:read"))
	if _, ok := quiet.AccessToken[ClaimApplicationRoles]; ok {
		t.Error("roles were carried without the roles scope")
	}
	if _, ok := quiet.IDToken["email"]; ok {
		t.Error("email was carried without the email scope")
	}
}

// A machine-to-machine token: no user, so the application's allowance is the
// only limit, and there is no ID token.
func TestEvaluateTokenMachine(t *testing.T) {
	f := newFixture()
	r := f.request("openid orders:write orders:refund")
	r.User, r.Roles = nil, nil
	r.Application.Type = AppM2M
	r.Application.GrantTypes = []string{GrantClientCredentials}

	p := EvaluateToken(r)

	if !p.Issued || p.AccessToken["scope"] != "orders:write" || p.AccessToken["sub"] != "shop-web" {
		t.Errorf("preview = %+v", p)
	}
	if p.IDToken != nil {
		t.Error("a machine token came with an ID token")
	}
	if d := decision(p, "openid"); d.Granted {
		t.Errorf("openid = %+v, want refused with no user", d)
	}
}

func TestAPIValidate(t *testing.T) {
	good := func() API {
		return API{
			Name:             "Orders",
			Identifier:       "https://api.shop.com/orders",
			SigningAlgorithm: AlgRS256,
			Scopes:           []APIScope{{Name: "orders:read"}},
		}
	}

	tests := []struct {
		name   string
		change func(*API)
		want   string
	}{
		{name: "an API", change: func(*API) {}},
		{name: "a plain identifier", change: func(a *API) { a.Identifier = "orders-api" }},
		{name: "its own token lifetime", change: func(a *API) { a.TokenLifetime = 900 }},
		{
			name:   "a shared-secret algorithm",
			change: func(a *API) { a.SigningAlgorithm = "HS256" },
			want:   "signing_algorithm must be one of: RS256, PS256, ES256",
		},
		{
			name:   "a token lifetime of seconds",
			change: func(a *API) { a.TokenLifetime = 5 },
			want:   "token_lifetime must be between 60 and 86400 seconds, or 0 to use the application's",
		},
		{name: "no name", change: func(a *API) { a.Name = " " }, want: "name is required"},
		{name: "a spaced identifier", change: func(a *API) { a.Identifier = "orders api" }, want: "identifier must not contain spaces"},
		{
			name:   "a scope with capitals",
			change: func(a *API) { a.Scopes = []APIScope{{Name: "Orders:Read"}} },
			want:   `scopes: "Orders:Read" must be lower case words joined by : . - or _, such as orders:read`,
		},
		{
			name:   "an OpenID scope name",
			change: func(a *API) { a.Scopes = []APIScope{{Name: "email"}} },
			want:   `scopes: "email" is an OpenID Connect scope and cannot be an API scope`,
		},
		{
			name:   "a scope twice",
			change: func(a *API) { a.Scopes = []APIScope{{Name: "orders:read"}, {Name: "orders:read"}} },
			want:   `scopes: "orders:read" is listed twice`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := good()
			tt.change(&a)

			err := a.Validate()
			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nothing", err)
			case tt.want != "" && (err == nil || err.Error() != tt.want):
				t.Fatalf("Validate() = %v, want %q", err, tt.want)
			}
		})
	}
}

// A default scope is added without being asked for, by the same rules as a
// requested one, and a refused default is reported rather than dropped
// silently.
func TestEvaluateTokenDefaultScopes(t *testing.T) {
	f := newFixture()
	f.orders.Scopes[0].IsDefault = true // orders:read: allowed, and support grants it
	f.orders.Scopes[1].IsDefault = true // orders:write: allowed, but no role grants it

	p := EvaluateToken(f.request("openid"))

	if got := p.AccessToken["scope"]; got != "openid orders:read" {
		t.Errorf("scope = %q, want openid plus the grantable default", got)
	}

	read := decision(p, "orders:read")
	if !read.Granted || !read.IsDefault || read.Reason != "default scope: granted by support" {
		t.Errorf("read = %+v, want granted as a default", read)
	}

	write := decision(p, "orders:write")
	if write.Granted || !write.IsDefault || write.Reason != "default scope: none of the user's roles grant this scope" {
		t.Errorf("write = %+v, want refused and reported as a default", write)
	}

	// Asked for explicitly, a default scope is an ordinary request.
	asked := EvaluateToken(f.request("orders:read"))
	if d := decision(asked, "orders:read"); d.IsDefault {
		t.Errorf("requested default = %+v, want it not marked as added", d)
	}
}

func TestEvaluateTokenAPISettings(t *testing.T) {
	f := newFixture()
	r := f.request("offline_access orders:read")
	r.API.AllowOfflineAccess = false
	r.API.TokenLifetime = 900
	r.API.SigningAlgorithm = AlgES256

	p := EvaluateToken(r)

	if d := decision(p, "offline_access"); d.Granted || d.Reason != "https://api.shop.com/orders does not allow refresh tokens" {
		t.Errorf("offline_access = %+v, want refused by the API", d)
	}
	if p.AccessToken["exp"].(int64)-p.AccessToken["iat"].(int64) != 900 {
		t.Errorf("lifetime = %v, want the API's 900 seconds", p.AccessToken)
	}
	if p.AccessTokenHeader["alg"] != AlgES256 || p.AccessTokenHeader["typ"] != "at+jwt" {
		t.Errorf("header = %v, want ES256 and at+jwt", p.AccessTokenHeader)
	}

	// With no audience, the application's lifetime and the default algorithm.
	r.API, r.Authorized, r.Allowed = nil, false, nil
	plain := EvaluateToken(r)
	if plain.AccessToken["exp"].(int64)-plain.AccessToken["iat"].(int64) != 3600 || plain.AccessTokenHeader["alg"] != AlgRS256 {
		t.Errorf("no audience: %v %v", plain.AccessTokenHeader, plain.AccessToken)
	}
}

// A role of another application grants nothing in this application's tokens:
// neither its name nor the API scopes it carries, or whoever manages that
// application could hand out scopes here.
func TestEvaluateTokenIgnoresOtherApplicationsRoles(t *testing.T) {
	f := newFixture()

	blog := uuid.New()
	blogWriter := UserRole{Name: "writer", ApplicationID: &blog, APIScopes: []APIScope{f.write},
		ID: uuid.New()}

	req := f.request("openid roles orders:write")
	req.Roles = []UserRole{blogWriter}

	got := EvaluateToken(req)

	if d := decision(got, "orders:write"); d.Granted {
		t.Errorf("orders:write = %+v, want it refused: only the blog's role grants it", d)
	}
	if roles := got.AccessToken[ClaimApplicationRoles]; len(roles.([]string)) != 0 {
		t.Errorf("roles claim = %v, want none of the blog's roles", roles)
	}

	// The same grant on one of the shop's own roles counts.
	shopWriter := UserRole{Name: "writer", ApplicationID: &f.app.ID, APIScopes: []APIScope{f.write},
		ID: uuid.New()}
	req.Roles = []UserRole{shopWriter}

	if d := decision(EvaluateToken(req), "orders:write"); !d.Granted {
		t.Errorf("orders:write = %+v, want it granted by the shop's own role", d)
	}
}
