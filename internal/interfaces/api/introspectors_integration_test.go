package api

import (
	"net/http"
	"net/url"
	"testing"
)

// An API's own backend, listed by an administrator, may introspect the access
// tokens issued for that API whichever client they went to, so it can see a
// revocation. Being allowed to request tokens for the API is not enough, a
// public client may never be listed or introspect, and the list is the API's
// alone.
func TestLiveAListedClientMayIntrospectItsAPIsTokens(t *testing.T) {
	f := newOAuthFixture(t)

	b := f.s.browser()
	verifier, challenge := pkce(t)
	web := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	if web.status != http.StatusOK {
		t.Fatalf("exchange = %d %v", web.status, web.body)
	}
	introspect := func(clientID, secret, token string) tokenResult {
		return f.token("/oauth2/introspect", clientID, secret, url.Values{"token": {token}})
	}

	backendID, backendSecret, backendApp := f.register(map[string]any{"name": "Orders backend", "type": "m2m"})
	authorisedID, authorisedSecret, authorisedApp := f.register(map[string]any{"name": "Reports", "type": "m2m"})
	f.authorizeAPI(authorisedApp, "orders:read")

	// Before anyone is listed, every other client's answer is inactive.
	if got := introspect(backendID, backendSecret, web.str("access_token")); got.body["active"] != false || got.body["sub"] != nil {
		t.Errorf("the backend before it was listed = %v, want active false", got.body)
	}

	var listed struct {
		Introspectors []struct {
			ID           string `json:"id"`
			Confidential bool   `json:"confidential"`
			Listed       bool   `json:"listed"`
		} `json:"introspectors"`
	}
	f.super.must(http.StatusOK, http.MethodPut, "/apis/"+f.apiID+"/introspectors",
		map[string]any{"application_ids": []string{backendApp}}, &listed)
	for _, it := range listed.Introspectors {
		if it.Listed != (it.ID == backendApp) {
			t.Errorf("application %s listed = %v after listing only the backend", it.ID, it.Listed)
		}
	}

	t.Run("a listed client may introspect", func(t *testing.T) {
		got := introspect(backendID, backendSecret, web.str("access_token"))
		if got.body["active"] != true || got.body["sub"] != f.userID || got.body["client_id"] != f.clientID {
			t.Errorf("the listed backend = %v, want the web application's token, active", got.body)
		}
		// Refresh tokens are their own client's alone: no API is handed one.
		if got := introspect(backendID, backendSecret, web.str("refresh_token")); got.body["active"] != false {
			t.Errorf("the listed backend on a refresh token = %v, want active false", got.body)
		}
	})

	t.Run("an unlisted authorised client gets active false", func(t *testing.T) {
		got := introspect(authorisedID, authorisedSecret, web.str("access_token"))
		if got.body["active"] != false || got.body["sub"] != nil {
			t.Errorf("a client authorised for the API but not listed = %v, want active false and nothing else", got.body)
		}
	})

	t.Run("a public client is refused", func(t *testing.T) {
		publicID, _, publicApp := f.register(map[string]any{
			"name": "Phone app", "type": "native", "grant_types": []string{"authorization_code"},
			"redirect_uris": []string{"com.example.app:/callback"},
		})

		var refused struct {
			Code string `json:"code"`
		}
		status := f.super.do(http.MethodPut, "/apis/"+f.apiID+"/introspectors",
			map[string]any{"application_ids": []string{backendApp, publicApp}}, &refused)
		if status != http.StatusBadRequest || refused.Code != "api_introspector_public" {
			t.Errorf("listing a public client = %d %s, want 400 api_introspector_public", status, refused.Code)
		}
		if got := introspect(publicID, "", web.str("access_token")); got.str("error") != "invalid_client" {
			t.Errorf("a public client introspecting = %d %v, want invalid_client", got.status, got.body)
		}
		// The refusal changed nothing: the backend is still the one listed.
		if got := introspect(backendID, backendSecret, web.str("access_token")); got.body["active"] != true {
			t.Errorf("the backend after a refused change = %v, want it still listed", got.body)
		}
	})

	t.Run("a revoked token reads as inactive to the listed client", func(t *testing.T) {
		if got := f.token("/oauth2/revoke", f.clientID, f.clientSecret, url.Values{"token": {web.str("access_token")}}); got.status != http.StatusOK {
			t.Fatalf("revoking = %d %v", got.status, got.body)
		}
		if got := introspect(backendID, backendSecret, web.str("access_token")); got.body["active"] != false {
			t.Errorf("the backend on a revoked token = %v, want active false", got.body)
		}
	})

	t.Run("taken off the list, it may not", func(t *testing.T) {
		fresh := f.exchange(func() (string, string) {
			v, c := pkce(t)
			return f.signIn(f.s.browser(), c).Query().Get("code"), v
		}())
		f.super.must(http.StatusOK, http.MethodPut, "/apis/"+f.apiID+"/introspectors",
			map[string]any{"application_ids": []string{}}, nil)
		if got := introspect(backendID, backendSecret, fresh.str("access_token")); got.body["active"] != false {
			t.Errorf("the backend once taken off = %v, want active false", got.body)
		}
	})

	var logs struct {
		Logs []struct {
			Action string `json:"action"`
		} `json:"logs"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/apis/"+f.apiID+"/logs", nil, &logs)
	changes := 0
	for _, entry := range logs.Logs {
		if entry.Action == "api.introspectors_updated" {
			changes++
		}
	}
	if changes != 2 {
		t.Errorf("the API's log records %d changes to its introspectors, want the 2 that were made", changes)
	}
}
