package api

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"

	"vard/internal/config"
)

// These tests put a reverse proxy in front of the server, as a deployment
// does, and check what the server believes about who is calling.
//
// The proxy either throws away any X-Forwarded-For the client sent and writes
// the address it was connected from, or, as many proxies do, appends that
// address to whatever the client
// sent. Every test client
// connects from 127.0.0.1, so a test names the address it stands for in
// testClientHeader and the proxy forwards that instead.

const testClientHeader = "X-Test-Client"

// behindProxy starts a server whose issuer is a reverse proxy in front of it,
// and returns the proxy's address.
func behindProxy(t *testing.T, appends bool, change func(*config.Config)) (*liveServer, string) {
	t.Helper()

	var target *url.URL
	proxy := &httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) {
		r.SetURL(target)
		r.Out.Host = r.In.Host

		client := r.In.Header.Get(testClientHeader)
		if client == "" {
			client = "127.0.0.1"
		}
		r.Out.Header.Del(testClientHeader)
		if sent := r.In.Header.Get("X-Forwarded-For"); appends && sent != "" {
			client = sent + ", " + client
		}
		r.Out.Header.Set("X-Forwarded-For", client)
		r.Out.Header.Set("X-Forwarded-Proto", "http")
	}}

	edge := httptest.NewUnstartedServer(proxy)
	edgeURL := "http://" + edge.Listener.Addr().String()

	s := newLiveServerWith(t, func(cfg *config.Config) {
		cfg.Issuer = edgeURL
		change(cfg)
	})

	var err error
	if target, err = url.Parse(s.root); err != nil {
		t.Fatal(err)
	}
	edge.Start()
	t.Cleanup(edge.Close)

	return s, edgeURL
}

var loopback = []string{"127.0.0.1/32", "::1/128"}

// viaProxy sends a JSON request to the proxy as `client`, with extra headers.
func viaProxy(t *testing.T, c *http.Client, method, target, body, client string, headers map[string]string) (int, []byte) {
	t.Helper()

	req, err := http.NewRequest(method, target, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(testClientHeader, client)
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	res, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	var raw json.RawMessage
	_ = json.NewDecoder(res.Body).Decode(&raw)

	return res.StatusCode, raw
}

const guess = `{"email":"nobody@example.com","password":"guess-password"}`

// The provider describes itself at the proxy's address, and a browser signs
// in through it: the cookie comes back, and the session records the address
// the browser really called from.
func TestLiveProxyServesTheProviderAtItsOwnAddress(t *testing.T) {
	s, edge := behindProxy(t, false, func(cfg *config.Config) { cfg.TrustedProxies = loopback })
	super := s.superAdmin()

	var discovery map[string]any
	if status, raw := viaProxy(t, http.DefaultClient, http.MethodGet, edge+"/.well-known/openid-configuration", "", "198.51.100.7", nil); status != http.StatusOK {
		t.Fatalf("discovery through the proxy = %d", status)
	} else if err := json.Unmarshal(raw, &discovery); err != nil {
		t.Fatal(err)
	}
	if discovery["issuer"] != edge || !strings.HasPrefix(discovery["token_endpoint"].(string), edge) {
		t.Errorf("discovery = issuer %v, token_endpoint %v; want both at %s", discovery["issuer"], discovery["token_endpoint"], edge)
	}

	const email, password = "grace@example.com", "grace-password-1"
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": email, "first_name": "Grace", "password": password, "confirm_password": password, "is_email_verified": true,
	}, nil)

	jar, _ := cookiejar.New(nil)
	browser := &http.Client{Jar: jar}
	login := `{"email":"` + email + `","password":"` + password + `"}`
	if status, raw := viaProxy(t, browser, http.MethodPost, edge+"/api/v1/account/login", login, "198.51.100.7", nil); status != http.StatusOK {
		t.Fatalf("sign-in through the proxy = %d %s", status, raw)
	}

	status, raw := viaProxy(t, browser, http.MethodGet, edge+"/api/v1/account/sessions", "", "198.51.100.7", nil)
	if status != http.StatusOK {
		t.Fatalf("sessions through the proxy = %d %s", status, raw)
	}
	var sessions struct {
		Sessions []struct {
			IP string `json:"ip"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(raw, &sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions.Sessions) != 1 || sessions.Sessions[0].IP != "198.51.100.7" {
		t.Errorf("sessions = %+v, want one from 198.51.100.7", sessions.Sessions)
	}
}

func TestLiveProxyRateLimit(t *testing.T) {
	for _, tt := range []struct {
		name    string
		appends bool
		trusted []string
		// forged is what each client writes into its own X-Forwarded-For.
		forged func(attempt int) string
		// otherClient is what a different client is answered after the first
		// has spent its budget.
		otherClient int
	}{
		{
			name:        "a trusted proxy gives each client its own budget",
			trusted:     loopback,
			forged:      func(int) string { return "" },
			otherClient: http.StatusUnauthorized,
		},
		{
			name:        "a forged X-Forwarded-For is dropped by a proxy that replaces it",
			trusted:     loopback,
			forged:      func(attempt int) string { return "203.0.113." + string(rune('1'+attempt)) },
			otherClient: http.StatusUnauthorized,
		},
		{
			// The server reads the chain from the right and stops at the
			// first address it does not trust: the client's real one.
			name:        "a forged X-Forwarded-For is ignored behind a proxy that appends",
			appends:     true,
			trusted:     loopback,
			forged:      func(attempt int) string { return "203.0.113." + string(rune('1'+attempt)) },
			otherClient: http.StatusUnauthorized,
		},
		{
			// Misconfigured, the limit fails closed: every client is the
			// proxy, sharing its one budget, rather than none being limited.
			name:        "an untrusted proxy is one client",
			trusted:     nil,
			forged:      func(int) string { return "" },
			otherClient: http.StatusTooManyRequests,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, edge := behindProxy(t, tt.appends, func(cfg *config.Config) {
				cfg.RateLimit = 2
				cfg.TrustedProxies = tt.trusted
			})
			login := edge + "/api/v1/account/login"

			for attempt := range 3 {
				headers := map[string]string{}
				if forged := tt.forged(attempt); forged != "" {
					headers["X-Forwarded-For"] = forged
				}

				want := http.StatusUnauthorized
				if attempt == 2 {
					want = http.StatusTooManyRequests
				}
				if status, raw := viaProxy(t, http.DefaultClient, http.MethodPost, login, guess, "198.51.100.1", headers); status != want {
					t.Fatalf("attempt %d = %d %s, want %d", attempt+1, status, raw, want)
				}
			}

			if status, _ := viaProxy(t, http.DefaultClient, http.MethodPost, login, guess, "198.51.100.2", nil); status != tt.otherClient {
				t.Errorf("another client = %d, want %d", status, tt.otherClient)
			}
		})
	}
}

// With nothing trusted, a caller reaching the server directly cannot choose
// its address either.
func TestLiveForwardedForIsIgnoredWithoutAProxy(t *testing.T) {
	s := newLiveServerLimited(t, 2)

	for attempt := range 3 {
		want := http.StatusUnauthorized
		if attempt == 2 {
			want = http.StatusTooManyRequests
		}
		headers := map[string]string{"Content-Type": "application/json", "X-Forwarded-For": "203.0.113." + string(rune('1'+attempt))}
		if status := raw(t, http.MethodPost, s.root+"/api/v1/account/login", guess, headers); status != want {
			t.Fatalf("attempt %d = %d, want %d", attempt+1, status, want)
		}
	}
}
