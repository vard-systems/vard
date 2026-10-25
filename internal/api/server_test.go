package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"vard/internal/config"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// testEngine builds the real public server, with its logging discarded.
func testEngine(origins ...string) *gin.Engine {
	cfg := config.Config{Addr: ":0", CORSOrigins: origins, AccountURL: "http://localhost:5173"}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	r, err := NewPublic(cfg, log, nil, nil)
	if err != nil {
		panic(err)
	}

	return r
}

// testAdminEngine builds the real admin server.
func testAdminEngine() *gin.Engine {
	cfg := config.Config{AdminURL: "http://localhost:5173"}

	r, err := NewAdmin(cfg, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	if err != nil {
		panic(err)
	}

	return r
}

// A path exists on one server only: the admin API is never on the public
// listener, and the provider never on the admin one.
func TestServersKeepTheirRoutesApart(t *testing.T) {
	public, admin := testEngine(), testAdminEngine()

	for _, path := range []string{"/api/v1/admin/me", "/api/v1/admin/setup", "/api/v1/admin/users"} {
		if w := do(public, http.MethodGet, path, nil); w.Code != http.StatusNotFound {
			t.Errorf("public GET %s = %d, want 404", path, w.Code)
		}
	}

	for _, path := range []string{"/.well-known/openid-configuration", "/oauth2/authorize", "/api/v1/account/me"} {
		if w := do(admin, http.MethodGet, path, nil); w.Code != http.StatusNotFound {
			t.Errorf("admin GET %s = %d, want 404", path, w.Code)
		}
	}

	for _, r := range []*gin.Engine{public, admin} {
		if w := do(r, http.MethodGet, "/healthz", nil); w.Code != http.StatusOK {
			t.Errorf("healthz = %d on one of the servers", w.Code)
		}
	}
}

// do sends a request through the router and returns the response.
func do(r *gin.Engine, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		path     string
		wantCode int
		wantBody map[string]string
	}{
		{
			name: "health check", method: http.MethodGet, path: "/healthz",
			wantCode: http.StatusOK, wantBody: map[string]string{"status": "ok"},
		},
		{
			name: "unknown path", method: http.MethodGet, path: "/nope",
			wantCode: http.StatusNotFound, wantBody: map[string]string{"code": "not_found", "error": "There is nothing here."},
		},
		{
			name: "unknown path under the api group", method: http.MethodGet, path: "/api/v1/nope",
			wantCode: http.StatusNotFound, wantBody: map[string]string{"code": "not_found", "error": "There is nothing here."},
		},
	}

	r := testEngine()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := do(r, tt.method, tt.path, nil)

			if w.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d (body: %s)", w.Code, tt.wantCode, w.Body)
			}

			var got map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatalf("body is not JSON: %s", w.Body)
			}
			for key, want := range tt.wantBody {
				if got[key] != want {
					t.Errorf("body[%q] = %q, want %q", key, got[key], want)
				}
			}
		})
	}
}

// The matrix of who may call the API lives with the middleware, in
// cors/cors_test.go. What matters here is that the engine actually
// mounts it: a server that forgot to would pass every test over there.
func TestEngineAppliesCORS(t *testing.T) {
	const allowed = "http://localhost:5173"

	r := testEngine(allowed)

	w := do(r, http.MethodGet, "/healthz", map[string]string{"Origin": allowed})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != allowed {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, allowed)
	}

	w = do(r, http.MethodGet, "/healthz", map[string]string{"Origin": "http://evil.test"})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it unset", got)
	}
}

// A caller cannot choose the address it is recorded under: with no proxy
// trusted, X-Forwarded-For is ignored.
func TestForwardedForIsNotTrustedByDefault(t *testing.T) {
	r := testEngine()

	var seen string
	r.GET("/ip", func(c *gin.Context) { seen = c.ClientIP() })

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "203.0.113.7:4000"
	req.Header.Set("X-Forwarded-For", "10.0.0.1")
	r.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "203.0.113.7" {
		t.Errorf("ClientIP() = %q, want the connection's own address", seen)
	}
}

func TestNewRefusesABadTrustedProxy(t *testing.T) {
	cfg := config.Config{TrustedProxies: []string{"not an address"}}

	if _, err := NewPublic(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil); err == nil {
		t.Error("New() accepted a trusted proxy that is not an address")
	}
}
