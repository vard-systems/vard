package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"

	"loginer/internal/webui"
)

var builtApp = fstest.MapFS{
	"index.html":                  {Data: []byte("<!doctype html><title>app</title>")},
	"_app/immutable/entry/app.js": {Data: []byte("export {}")},
}

// The app is served on every path the API does not answer, and only there:
// an unknown path under the API's own prefixes stays a JSON problem, and
// the API's routes are untouched.
func TestTheAppIsServedBesideTheAPI(t *testing.T) {
	public := testEngine()
	WithPublicUI(public, webui.SPA(builtApp))

	admin := testAdminEngine()
	WithAdminUI(admin, webui.SPA(builtApp))

	tests := []struct {
		name   string
		engine *gin.Engine
		method string
		path   string
		status int
		app    bool
	}{
		{"the sign-in page", public, http.MethodGet, "/login", http.StatusOK, true},
		{"a deep account route", public, http.MethodGet, "/security", http.StatusOK, true},
		{"the root", public, http.MethodGet, "/", http.StatusOK, true},
		{"a built file", public, http.MethodGet, "/_app/immutable/entry/app.js", http.StatusOK, false},
		{"an API route", public, http.MethodGet, "/healthz", http.StatusOK, false},
		{"an unknown account API path", public, http.MethodGet, "/api/v1/account/nothing", http.StatusNotFound, false},
		{"an unknown API path", public, http.MethodGet, "/api/does-not-exist", http.StatusNotFound, false},
		{"the API prefix itself", public, http.MethodGet, "/api", http.StatusNotFound, false},
		{"an unknown provider path", public, http.MethodGet, "/oauth2/nothing", http.StatusNotFound, false},
		{"an unknown discovery path", public, http.MethodGet, "/.well-known/nothing", http.StatusNotFound, false},
		{"a POST to a page", public, http.MethodPost, "/login", http.StatusNotFound, false},
		{"the admin API is not on the public listener", public, http.MethodGet, "/api/v1/admin/me", http.StatusNotFound, false},
		{"a panel route", admin, http.MethodGet, "/admin/dashboard/users", http.StatusOK, true},
		{"an unknown admin API path", admin, http.MethodGet, "/api/v1/admin/nothing", http.StatusNotFound, false},
		{"a DELETE to a page", admin, http.MethodDelete, "/admin/dashboard", http.StatusNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			tt.engine.ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))

			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d (body %q)", w.Code, tt.status, w.Body.String())
			}

			page := strings.Contains(w.Body.String(), "<title>app</title>")
			if page != tt.app {
				t.Errorf("served the app = %v, want %v (body %q)", page, tt.app, w.Body.String())
			}
			if tt.status == http.StatusNotFound && !strings.HasPrefix(w.Header().Get("Content-Type"), "application/") {
				t.Errorf("a 404 here is a JSON problem; Content-Type = %q", w.Header().Get("Content-Type"))
			}
		})
	}
}
