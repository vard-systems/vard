package api

import (
	"net/http"
	"testing"

	"vard/internal/model"
)

// adminCLI is the id of the application the server made for itself at
// startup.
func (f *oauthFixture) adminCLI() string {
	f.s.t.Helper()

	var list struct {
		Applications []struct {
			ID       string `json:"id"`
			ClientID string `json:"client_id"`
		} `json:"applications"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/applications?search="+model.AdminCLIClientID, nil, &list)
	for _, app := range list.Applications {
		if app.ClientID == model.AdminCLIClientID {
			return app.ID
		}
	}

	f.s.t.Fatal("admin-cli was not made at startup")
	return ""
}

// The power of an application the admin API trusts is in its secret: whoever
// holds it holds every scope a super admin allowed it. So an administrator
// with applications.write for that one application — who may not give it
// access to the admin API — may not rotate its secret, change it, or take
// that access away either.
func TestLiveOnlyASuperAdminChangesAnApplicationTheAdminAPITrusts(t *testing.T) {
	f := newOAuthFixture(t)
	cliID := f.adminCLI()

	apiID, scopes := f.systemAPI(model.SystemAdminAPI)
	f.allow(cliID, apiID, scopes, model.PermUsersRead)

	manager := f.s.appManager(f.super, cliID)

	var current struct {
		Application map[string]any `json:"application"`
	}
	manager.must(http.StatusOK, http.MethodGet, "/applications/"+cliID, nil, &current)

	tests := []struct {
		name         string
		method, path string
		body         any
		code         string
	}{
		{name: "rotating its secret", method: http.MethodPost, path: "/applications/" + cliID + "/secret", code: "system_application_change"},
		{name: "changing it", method: http.MethodPatch, path: "/applications/" + cliID, body: current.Application, code: "system_application_change"},
		{name: "taking the admin API away from it", method: http.MethodDelete, path: "/applications/" + cliID + "/apis/" + apiID, code: "system_api_access"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var refused problemBody
			if status := manager.do(tt.method, tt.path, tt.body, &refused); status != http.StatusForbidden || refused.Code != tt.code {
				t.Errorf("%s %s by an application's manager = %d %+v, want 403 %s", tt.method, tt.path, status, refused, tt.code)
			}
		})
	}

	// A super admin does all three.
	var rotated struct {
		ClientSecret string `json:"client_secret"`
	}
	f.super.must(http.StatusOK, http.MethodPost, "/applications/"+cliID+"/secret", nil, &rotated)
	if rotated.ClientSecret == "" {
		t.Error("a super admin rotated the secret and was not given it")
	}
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+cliID, current.Application, nil)
	f.super.must(http.StatusOK, http.MethodDelete, "/applications/"+cliID+"/apis/"+apiID, nil, nil)

	// No longer trusted by the admin API, the application is its manager's
	// to change again.
	manager.must(http.StatusOK, http.MethodPost, "/applications/"+cliID+"/secret", nil, &rotated)
}
