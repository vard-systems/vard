package applications

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"loginer/internal/api/respond"
	"loginer/internal/model"
)

func TestApplicationRequestApplyTo(t *testing.T) {
	good := func() applicationRequest {
		return applicationRequest{
			Name:         "  Shop ",
			Type:         "web",
			GrantTypes:   []string{"authorization_code", "refresh_token"},
			RedirectURIs: []string{"https://shop.example.com/callback"},
			Scopes:       []string{"openid", "email"},
		}
	}

	tests := []struct {
		name     string
		creating bool
		change   func(*applicationRequest)
		want     string // the message, or "" when the request is fine
	}{
		{name: "a web app", creating: true, change: func(*applicationRequest) {}},
		{name: "no name", creating: true, change: func(r *applicationRequest) { r.Name = " " }, want: "name is required."},
		{
			name:     "no type",
			creating: true,
			change:   func(r *applicationRequest) { r.Type = "" },
			want:     "The type must be one of: web, spa, native, m2m.",
		},
		{
			name:   "an update ignores the type",
			change: func(r *applicationRequest) { r.Type = "" },
		},
		{
			name:     "a redirect URI too long",
			creating: true,
			change: func(r *applicationRequest) {
				r.RedirectURIs = []string{"https://shop.example.com/" + strings.Repeat("a", 600)}
			},
			want: "redirect_uris[0] must be at most 512 characters.",
		},
		{
			name:     "too many redirect URIs",
			creating: true,
			change: func(r *applicationRequest) {
				for i := range 21 {
					r.RedirectURIs = append(r.RedirectURIs, "https://shop.example.com/"+strings.Repeat("a", i+1))
				}
			},
			want: "An application may have at most 20 redirect URIs of each kind.",
		},
		{
			name:     "a rule the model keeps",
			creating: true,
			change:   func(r *applicationRequest) { r.RedirectURIs = []string{"http://shop.example.com/cb"} },
			want:     `redirect_uris: "http://shop.example.com/cb" must use https, unless it is on localhost`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := good()
			tt.change(&request)

			app := &model.Application{Type: model.AppWeb}
			if tt.creating {
				app = &model.Application{Type: model.ApplicationType(request.Type)}
			}

			err := request.applyTo(app, tt.creating)

			if tt.want == "" {
				if err != nil {
					t.Fatalf("applyTo() = %v, want nothing", err)
				}
				if app.Name != "Shop" || app.AccessTokenLifetime != model.DefaultAccessTokenLifetime {
					t.Errorf("name = %q, access lifetime = %d; want it tidied and defaulted", app.Name, app.AccessTokenLifetime)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) {
				t.Fatalf("applyTo() = %v, want a respond.Fault", err)
			}
			if fault.Status != http.StatusBadRequest || fault.Message != tt.want {
				t.Errorf("fault = %d %q, want 400 %q", fault.Status, fault.Message, tt.want)
			}
		})
	}
}

// Switching a confidential client to a public method is not possible, since
// the type decides it; but a secret must not survive a client losing its
// secret method, however that came about.
func TestApplyToDropsASecretWithNoMethod(t *testing.T) {
	app := &model.Application{Type: model.AppSPA, ClientSecretHash: "abc", SecretHint: "wxyz"}

	request := applicationRequest{
		Name:         "SPA",
		GrantTypes:   []string{"authorization_code"},
		RedirectURIs: []string{"https://spa.example.com/cb"},
	}

	if err := request.applyTo(app, false); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	if app.ClientSecretHash != "" || app.SecretHint != "" || app.TokenAuthMethod != model.AuthNone {
		t.Errorf("hash = %q, hint = %q, method = %s; want no secret", app.ClientSecretHash, app.SecretHint, app.TokenAuthMethod)
	}
}

// A flag left out of a request keeps what the application had — on a create,
// the defaults it was made with — and a flag sent is taken as sent.
func TestApplyToFlags(t *testing.T) {
	off := false
	request := applicationRequest{
		Name:         "Shop",
		Type:         "web",
		GrantTypes:   []string{"authorization_code"},
		RedirectURIs: []string{"https://shop.example.com/cb"},
		IsEnabled:    &off,
	}

	app := &model.Application{Type: model.AppWeb, IsEnabled: true, AssertRoles: true, RequirePKCE: true}
	if err := request.applyTo(app, true); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	if app.IsEnabled || !app.AssertRoles || !app.RequirePKCE {
		t.Errorf("enabled = %v, assert roles = %v, pkce = %v; want sent false, the rest kept", app.IsEnabled, app.AssertRoles, app.RequirePKCE)
	}
}
