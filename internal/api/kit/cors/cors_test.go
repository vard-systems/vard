package cors

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const panelOrigin = "http://localhost:5173"

// corsEngine is a router with nothing on it but CORS and one route to reach.
func corsEngine(origins ...string) *gin.Engine {
	r := gin.New()
	r.Use(New(origins))
	r.GET("/thing", func(c *gin.Context) { c.Status(http.StatusOK) })

	return r
}

// send makes a request with the given origin, or none when it is empty.
func send(r *gin.Engine, method, origin string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/thing", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	return w
}

// TestCORS covers who is allowed to call the API from a browser. Getting this
// wrong is not a broken page: it is handing another site the ability to act
// as whoever is signed in, so every case is spelled out.
func TestCORS(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		origin  string
		method  string
		want    string // the origin we expect to be allowed, or "" for none
		status  int
	}{
		{
			name:    "an origin on the list",
			origins: []string{panelOrigin},
			origin:  panelOrigin,
			method:  http.MethodGet,
			want:    panelOrigin,
			status:  http.StatusOK,
		},
		{
			name:    "one of several",
			origins: []string{"http://localhost:4173", panelOrigin},
			origin:  panelOrigin,
			method:  http.MethodGet,
			want:    panelOrigin,
			status:  http.StatusOK,
		},
		{
			name:    "an origin that is not",
			origins: []string{panelOrigin},
			origin:  "http://evil.test",
			method:  http.MethodGet,
			status:  http.StatusOK,
		},
		{
			name:    "no origin at all",
			origins: []string{panelOrigin},
			method:  http.MethodGet,
			status:  http.StatusOK,
		},
		{
			name:   "nothing is allowed",
			origin: panelOrigin,
			method: http.MethodGet,
			status: http.StatusOK,
		},
		{
			name:    "a preflight from an allowed origin",
			origins: []string{panelOrigin},
			origin:  panelOrigin,
			method:  http.MethodOptions,
			want:    panelOrigin,
			status:  http.StatusNoContent,
		},
		{
			name:    "a preflight from anywhere else",
			origins: []string{panelOrigin},
			origin:  "http://evil.test",
			method:  http.MethodOptions,
			status:  http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := send(corsEngine(tt.origins...), tt.method, tt.origin)

			if w.Code != tt.status {
				t.Errorf("status = %d, want %d", w.Code, tt.status)
			}

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.want {
				t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, tt.want)
			}
		})
	}
}

// An allowed origin is told it may send the session cookie, and what it may
// send with it. Without the credentials header the panel's cookie would be
// dropped by the browser and every call would look signed out.
func TestCORSAllowsCredentialsAndTheMethodsWeUse(t *testing.T) {
	w := send(corsEngine(panelOrigin), http.MethodGet, panelOrigin)

	want := map[string]string{
		"Access-Control-Allow-Credentials": "true",
		"Access-Control-Allow-Methods":     allowedMethods,
		"Access-Control-Allow-Headers":     allowedHeaders,
		"Vary":                             "Origin",
	}

	for header, value := range want {
		if got := w.Header().Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
}

// A request that was refused gets none of those headers either, rather than
// being told what it may not do.
func TestCORSTellsAStrangerNothing(t *testing.T) {
	w := send(corsEngine(panelOrigin), http.MethodGet, "http://evil.test")

	for _, header := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Credentials",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
	} {
		if got := w.Header().Get(header); got != "" {
			t.Errorf("%s = %q, want it unset", header, got)
		}
	}
}

// An empty entry in the list — a stray comma in LOGINER_CORS_ORIGINS — must
// not turn every request without an Origin header into an allowed one.
func TestCORSIgnoresAnEmptyEntry(t *testing.T) {
	r := corsEngine("", panelOrigin)

	w := send(r, http.MethodGet, "")

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it unset", got)
	}
}

// Origins are compared exactly: a different port or scheme is a different
// origin, and a site that merely starts the same is a stranger.
func TestCORSComparesExactly(t *testing.T) {
	strangers := []string{
		"http://localhost:5174",
		"https://localhost:5173",
		"http://localhost:5173.evil.test",
		"http://localhost:5173/",
		"HTTP://LOCALHOST:5173",
	}

	for _, origin := range strangers {
		t.Run(origin, func(t *testing.T) {
			w := send(corsEngine(panelOrigin), http.MethodGet, origin)

			if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
				t.Errorf("Access-Control-Allow-Origin = %q, want it unset", got)
			}
		})
	}
}

// A preflight is answered by the middleware and never reaches the route.
func TestCORSAnswersThePreflightItself(t *testing.T) {
	reached := false

	r := gin.New()
	r.Use(New([]string{panelOrigin}))
	r.OPTIONS("/thing", func(c *gin.Context) { reached = true })

	send(r, http.MethodOptions, panelOrigin)

	if reached {
		t.Error("the preflight reached the route, want it answered by the middleware")
	}
}

// The provider endpoints a single-page app calls are open to any origin, and
// never with credentials; the admin API stays closed to the same origin.
func TestCORSPublicProviderEndpoints(t *testing.T) {
	r := gin.New()
	r.Use(New([]string{panelOrigin}))
	r.POST("/oauth2/token", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/.well-known/jwks.json", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/oauth2/authorize", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/api/v1/admin/me", func(c *gin.Context) { c.Status(http.StatusOK) })

	call := func(method, path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Origin", "https://spa.example.com")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	for _, path := range []string{"/oauth2/token", "/.well-known/jwks.json"} {
		method := http.MethodGet
		if path == "/oauth2/token" {
			method = http.MethodPost
		}

		w := call(method, path)
		if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
			t.Errorf("%s: Allow-Origin = %q, want *", path, got)
		}
		if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
			t.Errorf("%s: Allow-Credentials = %q, want none", path, got)
		}
	}

	if w := call(http.MethodOptions, "/oauth2/token"); w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("token preflight = %d, %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}

	for _, path := range []string{"/oauth2/authorize", "/api/v1/admin/me"} {
		if got := call(http.MethodGet, path).Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s: Allow-Origin = %q for an unlisted origin, want none", path, got)
		}
	}
}
