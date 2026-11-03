package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"vard/core/cache"
	"vard/core/cache/cachetest"
)

// The session database keeps administrators without their password, so a
// profile saved from a request signed in through it has to write back the
// password in the database, not an empty one: afterwards the administrator
// still signs in with it, and a new one still works.
func TestLiveAProfileSavedFromTheSessionCacheKeepsThePassword(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	// Twice, so the second request is answered from Redis when there is one.
	super.must(http.StatusOK, http.MethodGet, "/me", nil, nil)
	super.must(http.StatusOK, http.MethodPatch, "/me", map[string]string{
		"first_name": "Rooted", "email": superEmail,
	}, nil)

	again := s.client()
	if status := again.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("signing in after a profile save = %d, want the password kept", status)
	}

	super.must(http.StatusOK, http.MethodPost, "/me/password", map[string]string{
		"current_password": superPassword, "new_password": "root-password-2",
	}, nil)

	if status := s.client().login(superEmail, "root-password-2"); status != http.StatusOK {
		t.Errorf("signing in with the new password = %d, want 200", status)
	}
}

// Whatever a request was answered from, a revoked session and a changed role
// take effect on the next request.
func TestLiveChangesReachTheNextRequest(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	laptop := s.client()
	if status := laptop.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("second sign-in = %d", status)
	}
	laptop.must(http.StatusOK, http.MethodGet, "/me", nil, nil)
	laptop.must(http.StatusOK, http.MethodGet, "/me", nil, nil)

	// Signing out everywhere else ends the laptop's session at once.
	super.must(http.StatusOK, http.MethodDelete, "/sessions", nil, nil)
	if status := laptop.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("a session ended elsewhere = %d, want 401", status)
	}

	// Signing out ends this one.
	super.must(http.StatusOK, http.MethodGet, "/me", nil, nil)
	super.must(http.StatusOK, http.MethodPost, "/auth/logout", nil, nil)
	if status := super.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
		t.Errorf("a session after signing out = %d, want 401", status)
	}
}

type cacheKey struct {
	Name       string          `json:"name"`
	Kind       string          `json:"kind"`
	Group      string          `json:"group"`
	TTLSeconds int64           `json:"ttl_seconds"`
	Editable   bool            `json:"editable"`
	Value      json.RawMessage `json:"value"`
}

// A super admin sees both databases, can read and edit what the cache holds
// and clear it, cannot edit what decides what the signed-in may do, and
// flushing that signs nobody out. No session is ever kept in Redis.
func TestLiveCacheManagement(t *testing.T) {
	s := newLiveServer(t)
	if s.cache == nil {
		t.Skip("no test Redis: set " + cachetest.Env)
	}
	super := s.superAdmin()

	// Something reads the organisation, so it is cached.
	super.must(http.StatusOK, http.MethodGet, "/organization", nil, nil)

	var overview struct {
		Configured bool `json:"configured"`
		Databases  []struct {
			Name      string `json:"name"`
			Available bool   `json:"available"`
			Editable  bool   `json:"editable"`
		} `json:"databases"`
	}
	super.must(http.StatusOK, http.MethodGet, "/cache", nil, &overview)
	if !overview.Configured || len(overview.Databases) != 2 {
		t.Fatalf("overview = %+v, want both databases", overview)
	}
	if cacheDB, sessions := overview.Databases[0], overview.Databases[1]; cacheDB.Name != "cache" || !cacheDB.Editable ||
		sessions.Name != "sessions" || sessions.Editable {
		t.Errorf("databases = %+v, want the editable cache and the read-only session database", overview.Databases)
	}

	var listing struct {
		Keys []cacheKey `json:"keys"`
	}
	super.must(http.StatusOK, http.MethodGet, "/cache/cache/keys?kind=entry&group=organization", nil, &listing)
	if len(listing.Keys) != 1 || !listing.Keys[0].Editable || listing.Keys[0].TTLSeconds <= 0 {
		t.Fatalf("keys = %+v, want the organisation's settings", listing.Keys)
	}
	name := url.QueryEscape(listing.Keys[0].Name)

	var read struct {
		Key cacheKey `json:"key"`
	}
	super.must(http.StatusOK, http.MethodGet, "/cache/cache/key?name="+name, nil, &read)
	if !strings.Contains(string(read.Key.Value), `"slug"`) {
		t.Errorf("value = %s, want the organisation", read.Key.Value)
	}

	super.must(http.StatusBadRequest, http.MethodPut, "/cache/cache/key?name="+name, map[string]any{"value": nil}, nil)
	super.must(http.StatusOK, http.MethodPut, "/cache/cache/key?name="+name, map[string]any{
		"value": json.RawMessage(read.Key.Value), "ttl_seconds": 60,
	}, &read)
	if read.Key.TTLSeconds > 60 {
		t.Errorf("ttl after writing = %d, want the minute asked for", read.Key.TTLSeconds)
	}

	super.must(http.StatusConflict, http.MethodDelete, "/cache/cache/key?name="+url.QueryEscape("cache:organization:generation"), nil, nil)
	super.must(http.StatusNoContent, http.MethodPost, "/cache/cache/groups/organization/clear", nil, nil)
	super.must(http.StatusNotFound, http.MethodPost, "/cache/cache/groups/admins/clear", nil, nil)
	super.must(http.StatusNotFound, http.MethodGet, "/cache/elsewhere/keys", nil, nil)

	// Sessions are read from Postgres every time: none is in Redis, though
	// this administrator is signed in.
	if held := s.cache.SessionDB(); held != nil {
		keys, _, err := held.Keys(context.Background(), cache.KeyQuery{Search: "session:"})
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) != 0 {
			t.Errorf("Redis holds %d session keys, want none", len(keys))
		}
	}

	// Nothing that decides what the signed-in may do is edited by hand: the
	// administrator this request read is cached, and stays as it is.
	super.must(http.StatusOK, http.MethodGet, "/cache/sessions/keys?kind=entry&group=admins", nil, &listing)
	if len(listing.Keys) == 0 {
		t.Fatal("the session database holds no administrator")
	}
	admin := url.QueryEscape(listing.Keys[0].Name)
	super.must(http.StatusConflict, http.MethodPut, "/cache/sessions/key?name="+admin, map[string]any{"value": map[string]any{}}, nil)

	// Flushing it signs nobody out.
	super.must(http.StatusNoContent, http.MethodDelete, "/cache/sessions", nil, nil)
	super.must(http.StatusOK, http.MethodGet, "/me", nil, nil)

	// An administrator who is not a super admin is kept out.
	manager := s.appManager(super, super.application("Shop"))
	manager.must(http.StatusForbidden, http.MethodGet, "/cache", nil, nil)
}
