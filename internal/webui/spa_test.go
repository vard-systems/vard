package webui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

var app = fstest.MapFS{
	"index.html":                  {Data: []byte("<!doctype html><title>app</title>")},
	"theme.js":                    {Data: []byte("// theme")},
	"_app/version.json":           {Data: []byte(`{"version":"1"}`)},
	"_app/immutable/entry/app.js": {Data: []byte("export {}")},
}

func TestSPA(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		path        string
		status      int
		body        string
		cache       string
		contentType string
	}{
		{"the root is the app", http.MethodGet, "/", http.StatusOK, "<title>app</title>", "no-cache", "text/html"},
		{"a deep route is the app", http.MethodGet, "/admin/dashboard/users", http.StatusOK, "<title>app</title>", "no-cache", "text/html"},
		{"a route with a query is the app", http.MethodGet, "/reset-password?token=x", http.StatusOK, "<title>app</title>", "no-cache", "text/html"},
		{"a hashed file is cached for a year", http.MethodGet, "/_app/immutable/entry/app.js", http.StatusOK, "export {}", "public, max-age=31536000, immutable", "javascript"},
		{"an unhashed file is revalidated", http.MethodGet, "/theme.js", http.StatusOK, "// theme", "no-cache", "javascript"},
		{"the version file is revalidated", http.MethodGet, "/_app/version.json", http.StatusOK, `"version"`, "no-cache", "application/json"},
		{"a missing chunk is a 404, not the app", http.MethodGet, "/_app/immutable/entry/gone.js", http.StatusNotFound, "", "", ""},
		{"a missing file is a 404, not the app", http.MethodGet, "/favicon.ico", http.StatusNotFound, "", "", ""},
		{"a directory is the app, never a listing", http.MethodGet, "/_app/immutable/", http.StatusOK, "<title>app</title>", "no-cache", "text/html"},
		{"a path climbing out is refused", http.MethodGet, "/../../etc/passwd", http.StatusBadRequest, "", "", ""},
		{"an encoded climb is refused", http.MethodGet, "/%2e%2e/%2e%2e/admin", http.StatusBadRequest, "", "", ""},
		{"HEAD is answered", http.MethodHead, "/", http.StatusOK, "", "no-cache", "text/html"},
		{"a POST is refused", http.MethodPost, "/", http.StatusMethodNotAllowed, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SPA(app).ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))

			if w.Code != tt.status {
				t.Fatalf("status = %d, want %d (body %q)", w.Code, tt.status, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), tt.body) {
				t.Errorf("body = %q, want it to contain %q", w.Body.String(), tt.body)
			}
			if tt.cache != "" && w.Header().Get("Cache-Control") != tt.cache {
				t.Errorf("Cache-Control = %q, want %q", w.Header().Get("Cache-Control"), tt.cache)
			}
			if !strings.Contains(w.Header().Get("Content-Type"), tt.contentType) {
				t.Errorf("Content-Type = %q, want %q", w.Header().Get("Content-Type"), tt.contentType)
			}
		})
	}
}

// The API's middleware runs first and sets a policy that forbids every
// script; the app's page has to replace it, keeping only what a <meta>
// cannot say.
func TestSPAReplacesTheAPIHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

	SPA(app).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/login", nil))

	want := map[string]string{
		"Content-Security-Policy": "frame-ancestors 'none'",
		"X-Frame-Options":         "DENY",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"Permissions-Policy":      permissionsPolicy,
	}
	for header, value := range want {
		if got := w.Header().Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
}
