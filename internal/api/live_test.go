package api

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"loginer/i18n"
	"loginer/internal/cache"
	"loginer/internal/cache/cachetest"
	"loginer/internal/config"
	"loginer/internal/database/databasetest"
	"loginer/internal/jose"
	"loginer/internal/mail"
	"loginer/internal/oidc"
	"loginer/internal/store"
)

// These tests run the whole server — routes, permission checks, store and
// migrations — against a real Postgres. They need a database server to make
// throwaway databases on, named by LOGINER_TEST_DB_DSN:
//
//	LOGINER_TEST_DB_DSN=postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable go test ./internal/api/
//
// Without it they are skipped, so `go test ./...` still passes anywhere.

// liveServer is the server running on a database of its own.
type liveServer struct {
	t   *testing.T
	url string
	// root is the public server's address: the issuer of its tokens.
	root string
	// adminRoot is the admin server's.
	adminRoot string
	mail      *mailbox
	// store is the server's own, for a test that checks what it keeps.
	store *store.Store
	// cache is the Redis the server reads through, or nil when the tests
	// have none.
	cache *cache.Redis
	db    *gorm.DB
	cfg   config.Config
}

// testAccountURL is where the provider sends browsers to sign in. Nothing is
// served there in the tests: they read the redirect and call the account
// endpoints the page would.
const testAccountURL = "http://account.test"

// mailbox keeps what the server would have emailed.
type mailbox struct {
	mu       sync.Mutex
	messages []mail.Message
}

func (m *mailbox) Send(_ context.Context, msg mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

// wait returns the first message sent to `to`, waiting briefly: the server
// sends in the background.
func (m *mailbox) wait(t *testing.T, to string) mail.Message {
	t.Helper()

	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		m.mu.Lock()
		for _, msg := range m.messages {
			if strings.EqualFold(msg.To, to) {
				m.mu.Unlock()
				return msg
			}
		}
		m.mu.Unlock()
	}

	t.Fatalf("no email to %s", to)
	return mail.Message{}
}

// linkIn is the link a message carries to one of the sign-in pages —
// "/verify-email", "/reset-password" — parsed. The body is plain text, so the
// link runs to the first space or newline after it.
func linkIn(t *testing.T, body, page string) *url.URL {
	t.Helper()

	at := strings.Index(body, testAccountURL+page)
	if at < 0 {
		t.Fatalf("no %s link in the message:\n%s", page, body)
	}

	link, err := url.Parse(strings.Fields(body[at:])[0])
	if err != nil {
		t.Fatal(err)
	}

	return link
}

// tokenIn is that link's token, which is what an endpoint is given to use it.
func tokenIn(t *testing.T, body, page string) string {
	t.Helper()

	token := linkIn(t, body, page).Query().Get("token")
	if token == "" {
		t.Fatalf("the %s link carries no token:\n%s", page, body)
	}

	return token
}

// newLiveServer makes an empty database, migrates it, and serves the API on
// it, without a rate limit. The database is dropped when the test ends.
func newLiveServer(t *testing.T) *liveServer {
	t.Helper()
	return newLiveServerLimited(t, 0)
}

// newLiveServerLimited is newLiveServer with a rate limit, per minute.
func newLiveServerLimited(t *testing.T, limit int) *liveServer {
	t.Helper()
	return newLiveServerWith(t, func(cfg *config.Config) { cfg.RateLimit = limit })
}

// newLiveServerWith is newLiveServer with the configuration changed first.
func newLiveServerWith(t *testing.T, change func(*config.Config)) *liveServer {
	t.Helper()

	db, cfg := databasetest.Open(t)

	return startNode(t, db, cfg, cachetest.Open(t), &mailbox{}, "", change, true)
}

// replica starts a second server on the same database and Redis, under the
// same issuer: what a load balancer in front of two instances looks like.
func (s *liveServer) replica() *liveServer {
	s.t.Helper()
	return startNode(s.t, s.db, s.cfg.DB, s.cache, s.mail, s.cfg.Issuer, func(cfg *config.Config) {
		*cfg = s.cfg
	}, false)
}

// startNode serves the public and admin APIs on their own listeners. The
// issuer is the public listener's address unless one is given. The first node
// seeds the database the way main does on a fresh installation.
func startNode(t *testing.T, db *gorm.DB, dbCfg config.DB, shared *cache.Redis, mailer *mailbox, issuer string, change func(*config.Config), first bool) *liveServer {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// The issuer has to be the address the server answers on, which is only
	// known once it has a listener: so the listeners come first, and the
	// routers are built to fit them.
	var publicRouter, adminRouter http.Handler
	publicServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		publicRouter.ServeHTTP(w, r)
	}))
	adminServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		adminRouter.ServeHTTP(w, r)
	}))
	root := "http://" + publicServer.Listener.Addr().String()
	adminRoot := "http://" + adminServer.Listener.Addr().String()
	if issuer == "" {
		issuer = root
	}

	serverCfg := config.Config{
		DB:         dbCfg,
		Issuer:     issuer,
		AccountURL: testAccountURL,
		AdminURL:   "http://admin.test",
		SecretKey:  "integration-test-secret-key-0123456789",
	}
	change(&serverCfg)

	// With LOGINER_TEST_REDIS the whole server runs with its cache, under a
	// prefix of its own, so every test here also proves that a write is seen
	// by the next read through the cache.
	st := store.New(db).WithCache(shared)
	if first {
		seed(t, st, serverCfg)
	}

	sealer, err := jose.NewSealer(serverCfg.SecretKey)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := oidc.New(context.Background(), serverCfg, st, sealer, mailer, log)
	if err != nil {
		t.Fatal(err)
	}
	publicEngine, err := NewPublic(serverCfg, log, provider, shared)
	if err != nil {
		t.Fatal(err)
	}
	adminEngine, err := NewAdmin(serverCfg, st, sealer, log, provider, shared)
	if err != nil {
		t.Fatal(err)
	}
	publicRouter, adminRouter = publicEngine, adminEngine

	publicServer.Start()
	adminServer.Start()
	t.Cleanup(publicServer.Close)
	t.Cleanup(adminServer.Close)

	return &liveServer{
		t: t, url: adminRoot + "/api/v1/admin", root: root, adminRoot: adminRoot,
		mail: mailer, store: st, cache: shared, db: db, cfg: serverCfg,
	}
}

// seed writes what main writes before serving: the administrators' sign-in
// settings, the shipped languages and the server's own APIs.
func seed(t *testing.T, st *store.Store, cfg config.Config) {
	t.Helper()

	ctx := context.Background()
	if err := st.EnsureAdminSecurity(ctx, cfg.AdminMFARequired); err != nil {
		t.Fatal(err)
	}
	shipped, err := i18n.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	if err := st.EnsureLanguages(ctx, shipped); err != nil {
		t.Fatal(err)
	}
	if err := st.EnsureSystemAPIs(ctx); err != nil {
		t.Fatal(err)
	}
}

// client is one browser: it keeps its own session cookie.
type client struct {
	s    *liveServer
	http *http.Client
}

func (s *liveServer) client() *client {
	jar, err := cookiejar.New(nil)
	if err != nil {
		s.t.Fatal(err)
	}

	return &client{s: s, http: &http.Client{Jar: jar}}
}

// do sends a JSON request and decodes the JSON answer into `out`, when given.
func (c *client) do(method, path string, body any, out any) int {
	c.s.t.Helper()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			c.s.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, c.s.url+path, reader)
	if err != nil {
		c.s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		c.s.t.Fatal(err)
	}
	defer res.Body.Close()

	if out != nil {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil && err != io.EOF {
			c.s.t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}

	return res.StatusCode
}

// must sends a request that has to answer with `want`.
func (c *client) must(want int, method, path string, body any, out any) {
	c.s.t.Helper()

	var raw json.RawMessage
	status := c.do(method, path, body, &raw)
	if status != want {
		c.s.t.Fatalf("%s %s = %d %s, want %d", method, path, status, raw, want)
	}

	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			c.s.t.Fatal(err)
		}
	}
}

func (c *client) login(email, password string) int {
	return c.do(http.MethodPost, "/auth/login", map[string]string{"username": email, "password": password}, nil)
}

const superEmail, superPassword = "root@example.com", "root-password-1"

// superAdmin sets the panel up and signs its first administrator in.
func (s *liveServer) superAdmin() *client {
	c := s.client()
	c.must(http.StatusCreated, http.MethodPost, "/setup", map[string]string{
		"email": superEmail, "password": superPassword, "first_name": "Root",
	}, nil)

	if status := c.login(superEmail, superPassword); status != http.StatusOK {
		s.t.Fatalf("super admin login = %d", status)
	}

	return c
}

type idOnly struct {
	ID string `json:"id"`
}

// application registers a web application and returns its id.
func (c *client) application(name string) string {
	var out struct {
		Application idOnly `json:"application"`
	}
	c.must(http.StatusCreated, http.MethodPost, "/applications", map[string]any{
		"name": name, "type": "web",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"https://" + name + ".example.com/callback"},
		"scopes":        []string{"openid"},
	}, &out)

	return out.Application.ID
}

// role creates a user role — global when app is "" — and returns its id.
func (c *client) role(name, app string, inherits ...string) string {
	body := map[string]any{"name": name, "inherits": inherits}
	if app != "" {
		body["application_id"] = app
	}

	var out struct {
		Role idOnly `json:"role"`
	}
	c.must(http.StatusCreated, http.MethodPost, "/user-roles", body, &out)

	return out.Role.ID
}

// adminRoleID finds a seeded admin role by name.
func (c *client) adminRoleID(name string) string {
	var out struct {
		Roles []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"roles"`
	}
	c.must(http.StatusOK, http.MethodGet, "/admin-roles", nil, &out)

	for _, role := range out.Roles {
		if role.Name == name {
			return role.ID
		}
	}

	c.s.t.Fatalf("no admin role %q", name)
	return ""
}

// appManager creates an administrator holding app_manager for one application
// and signs them in.
func (s *liveServer) appManager(super *client, app string) *client {
	const email, password = "manager@example.com", "manager-password-1"

	super.must(http.StatusCreated, http.MethodPost, "/admins", map[string]any{
		"email": email, "first_name": "Manager", "status": "active",
		"password": password, "confirm_password": password,
		"assignments": []map[string]any{{"role_id": super.adminRoleID("app_manager"), "application_id": app}},
	}, nil)

	c := s.client()
	if status := c.login(email, password); status != http.StatusOK {
		s.t.Fatalf("app manager login = %d", status)
	}

	return c
}

func randomHex(t *testing.T, n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}

	return hex.EncodeToString(b)
}

// defaultFlow is the id of the flow every application without one of its own
// falls back to.
func defaultFlow(t *testing.T, super *client) string {
	t.Helper()

	var flows struct {
		Flows []struct {
			ID        string `json:"id"`
			IsDefault bool   `json:"is_default"`
		} `json:"flows"`
	}
	super.must(http.StatusOK, http.MethodGet, "/login-flows", nil, &flows)

	for _, flow := range flows.Flows {
		if flow.IsDefault {
			return flow.ID
		}
	}

	t.Fatal("there is no default flow")

	return ""
}
