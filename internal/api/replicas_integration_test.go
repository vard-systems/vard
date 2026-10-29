package api

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"vard/internal/model"
)

// These tests run two servers on one database and one Redis, the way a
// deployment scales out behind a load balancer. Nothing a request needs may
// live in one process's memory: a token, a code, a session or a setting
// written through one instance has to hold on the other at once.

// on is the fixture with its requests sent to another instance.
func (f *oauthFixture) on(node *liveServer) *oauthFixture {
	moved := *f
	moved.s = node
	return &moved
}

func TestLiveReplicasShareTokensKeysAndSessions(t *testing.T) {
	f := newOAuthFixture(t)
	other := f.on(f.s.replica())

	b := f.s.browser()
	verifier, challenge := pkce(t)
	tokens := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	if tokens.status != http.StatusOK {
		t.Fatalf("exchange on the first instance = %d %v", tokens.status, tokens.body)
	}

	// The second instance verifies the first one's signature from the keys
	// they share, and refreshes the token the first one issued.
	other.verifyJWT(tokens.str("access_token"))

	introspected := other.token("/oauth2/introspect", f.clientID, f.clientSecret, url.Values{"token": {tokens.str("access_token")}})
	if introspected.body["active"] != true {
		t.Errorf("introspect on the second instance = %v", introspected.body)
	}

	refreshed := other.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tokens.str("refresh_token")},
	})
	if refreshed.status != http.StatusOK {
		t.Fatalf("refresh on the second instance = %d %v", refreshed.status, refreshed.body)
	}

	// The browser's session, started on the first, signs it in on the second.
	elsewhere := &browser{t: t, s: other.s, http: b.http}
	if status := elsewhere.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Fatalf("GET /me on the second instance = %d, want the session the first started", status)
	}

	// Signing out on the first ends it on the second at the next request.
	if status := b.account(http.MethodPost, "/logout", nil, nil); status >= 300 {
		t.Fatalf("logout = %d", status)
	}
	if status := elsewhere.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("GET /me on the second instance after signing out = %d, want 401", status)
	}
}

// One code, presented to both instances at once, is spent exactly once.
func TestLiveReplicasSpendACodeOnce(t *testing.T) {
	f := newOAuthFixture(t)
	nodes := []*oauthFixture{f, f.on(f.s.replica())}

	verifier, challenge := pkce(t)
	code := f.signIn(f.s.browser(), challenge).Query().Get("code")

	const attempts = 12
	statuses := make([]int, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			statuses[i] = nodes[i%2].exchange(code, verifier).status
		})
	}
	wg.Wait()

	if won := count(statuses, http.StatusOK); won != 1 {
		t.Errorf("%d of %d concurrent exchanges succeeded, want exactly one: %v", won, attempts, statuses)
	}
}

// One refresh token, presented to both instances at once, is one client
// retrying: it rotates once, every request is given that same new token, and
// once the grace is over the old one presented again is a replay that ends
// the family.
func TestLiveReplicasRotateARefreshTokenOnce(t *testing.T) {
	f := newOAuthFixture(t)
	nodes := []*oauthFixture{f, f.on(f.s.replica())}

	verifier, challenge := pkce(t)
	first := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier).str("refresh_token")

	const attempts = 12
	results := make([]tokenResult, attempts)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			results[i] = nodes[i%2].token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
				"grant_type": {"refresh_token"}, "refresh_token": {first},
			})
		})
	}
	wg.Wait()

	issued := map[string]bool{}
	for _, r := range results {
		if r.status != http.StatusOK {
			t.Errorf("concurrent refresh within the grace = %d %v, want tokens", r.status, r.body)
		}
		issued[r.str("refresh_token")] = true
	}
	if len(issued) != 1 {
		t.Errorf("%d different refresh tokens for %d refreshes, want the one rotation's", len(issued), attempts)
	}
	if live := f.s.liveRefreshTokens(f.userID); live != 1 {
		t.Fatalf("%d live refresh tokens, want exactly one", live)
	}

	f.s.agoRefreshRotations(model.RefreshReuseGrace)
	if replay := f.refresh(first); replay.status != http.StatusBadRequest {
		t.Errorf("the first token after the grace = %d %v, want a replay refused", replay.status, replay.body)
	}
	if live := f.s.liveRefreshTokens(f.userID); live != 0 {
		t.Errorf("%d live refresh tokens after a replay, want the family revoked", live)
	}
}

// A change made through one instance's admin API is what the other answers
// with at its next request, cache or no cache.
func TestLiveReplicasSeeEachOthersChanges(t *testing.T) {
	f := newOAuthFixture(t)
	other := f.on(f.s.replica())

	clientID, secret, appID := f.register(map[string]any{"name": "Nightly job", "type": "m2m"})
	credentials := url.Values{"grant_type": {"client_credentials"}, "audience": {ordersAPI}}
	f.authorizeAPI(appID, "orders:read")

	if r := other.token("/oauth2/token", clientID, secret, credentials); r.status != http.StatusOK {
		t.Fatalf("client credentials on the second instance = %d %v", r.status, r.body)
	}

	f.setEnabled(appID, false)

	if r := other.token("/oauth2/token", clientID, secret, credentials); r.status == http.StatusOK {
		t.Errorf("client credentials on the second instance after disabling = %d, want refused", r.status)
	}

	var rotated struct {
		ClientSecret string `json:"client_secret"`
	}
	f.setEnabled(appID, true)
	f.super.must(http.StatusOK, http.MethodPost, "/applications/"+appID+"/secret", nil, &rotated)

	if r := other.token("/oauth2/token", clientID, secret, credentials); r.status == http.StatusOK {
		t.Errorf("the old secret on the second instance after rotating = %d, want refused", r.status)
	}
	if r := other.token("/oauth2/token", clientID, rotated.ClientSecret, credentials); r.status != http.StatusOK {
		t.Errorf("the new secret on the second instance = %d %v", r.status, r.body)
	}
}

// What a token may carry is cached, so this is what proves the cache is
// forgotten: an API taken away from an application, or a scope taken away
// from a role, through one instance, is refused by the other at once.
func TestLiveReplicasSeeGrantChanges(t *testing.T) {
	f := newOAuthFixture(t)
	other := f.on(f.s.replica())

	clientID, secret, appID := f.register(map[string]any{"name": "Nightly job", "type": "m2m"})
	f.authorizeAPI(appID, "orders:read")
	credentials := url.Values{"grant_type": {"client_credentials"}, "audience": {ordersAPI}, "scope": {"orders:read"}}

	if r := other.token("/oauth2/token", clientID, secret, credentials); r.str("scope") != "orders:read" {
		t.Fatalf("client credentials on the second instance = %d %v", r.status, r.body)
	}
	f.super.must(http.StatusOK, http.MethodDelete, "/applications/"+appID+"/apis/"+f.apiID, nil, nil)
	if r := other.token("/oauth2/token", clientID, secret, credentials); r.status == http.StatusOK {
		t.Errorf("client credentials after the API was taken away = %d %v, want refused", r.status, r.body)
	}

	verifier, challenge := pkce(t)
	tokens := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier)
	if !strings.Contains(tokens.str("scope"), "orders:read") {
		t.Fatalf("exchange = %d %v, want orders:read from the customer role", tokens.status, tokens.body)
	}

	var roles struct {
		Roles []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"roles"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/user-roles", nil, &roles)
	for _, role := range roles.Roles {
		if role.Name == "customer" {
			f.super.must(http.StatusOK, http.MethodPatch, "/user-roles/"+role.ID, map[string]any{
				"name": "customer", "api_scopes": []string{},
			}, nil)
		}
	}

	refreshed := other.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {tokens.str("refresh_token")},
	})
	if refreshed.status != http.StatusOK || strings.Contains(refreshed.str("scope"), "orders:read") {
		t.Errorf("refresh on the second instance after the role lost the scope = %d %v, want no orders:read", refreshed.status, refreshed.body)
	}
}

// Wrong passwords sent to both instances at once count against one account,
// and the wait they set holds on both.
func TestLiveReplicasShareTheThrottle(t *testing.T) {
	s := newLiveServer(t)
	s.superAdmin()
	nodes := []*liveServer{s, s.replica()}

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			nodes[i%2].client().login(superEmail, "wrong-password")
		})
	}
	wg.Wait()

	for i, node := range nodes {
		if status := node.client().login(superEmail, superPassword); status != http.StatusUnauthorized {
			t.Errorf("right password on instance %d after the burst = %d, want the account waiting", i+1, status)
		}
	}
}

// With Redis, the rate limit is one budget per address across every
// instance, rather than one per instance.
func TestLiveReplicasShareTheRateLimit(t *testing.T) {
	s := newLiveServerLimited(t, 3)
	if s.cache == nil {
		t.Skip("the rate limit is shared through Redis, and these tests have none")
	}
	nodes := []*liveServer{s, s.replica()}

	body := `{"email":"nobody@example.com","password":"guess-password"}`
	json := map[string]string{"Content-Type": "application/json"}

	for i := range 3 {
		if status := raw(t, http.MethodPost, nodes[i%2].root+"/api/v1/account/login", body, json); status != http.StatusUnauthorized {
			t.Fatalf("attempt %d = %d, want 401 inside the limit", i+1, status)
		}
	}
	for i, node := range nodes {
		if status := raw(t, http.MethodPost, node.root+"/api/v1/account/login", body, json); status != http.StatusTooManyRequests {
			t.Errorf("instance %d past the shared limit = %d, want 429", i+1, status)
		}
	}
}

// setEnabled turns an application on or off the way the panel does: the
// update replaces the settings, so it sends back what it read.
func (f *oauthFixture) setEnabled(appID string, enabled bool) {
	var read struct {
		Application map[string]any `json:"application"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/applications/"+appID, nil, &read)

	read.Application["is_enabled"] = enabled
	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+appID, read.Application, nil)
}

func count(statuses []int, want int) int {
	n := 0
	for _, status := range statuses {
		if status == want {
			n++
		}
	}
	return n
}
