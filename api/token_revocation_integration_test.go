package api

import (
	"net/http"
	"net/url"
	"slices"
	"testing"
)

// Access tokens are JWTs an API checks on its own, but this server's own
// endpoints — userinfo and introspection — stop vouching for one as soon as
// what it was issued in ends: the session signed out, the application
// disconnected, the refresh family caught being replayed, or the token
// itself revoked.

// userinfoStatus is what userinfo answers an access token.
func (f *oauthFixture) userinfoStatus(token string) int {
	f.s.t.Helper()

	req, err := http.NewRequest(http.MethodGet, f.s.root+"/oauth2/userinfo", nil)
	if err != nil {
		f.s.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		f.s.t.Fatal(err)
	}
	res.Body.Close()

	return res.StatusCode
}

// active is what introspection says of a token.
func (f *oauthFixture) active(token string) bool {
	f.s.t.Helper()

	answer := f.token("/oauth2/introspect", f.clientID, f.clientSecret, url.Values{"token": {token}})
	return answer.body["active"] == true
}

func TestLiveAnAccessTokenEndsWithWhatItWasIssuedIn(t *testing.T) {
	cases := []struct {
		name string
		end  func(t *testing.T, f *oauthFixture, b *browser, tokens tokenResult)
	}{
		{"the session signs out", func(t *testing.T, f *oauthFixture, b *browser, tokens tokenResult) {
			b.visit(f.s.root + "/oauth2/logout?id_token_hint=" + url.QueryEscape(tokens.str("id_token")))
		}},
		{"the user disconnects the application", func(t *testing.T, f *oauthFixture, b *browser, _ tokenResult) {
			if status := b.account(http.MethodDelete, "/connected-applications/"+f.clientID, nil, nil); status != http.StatusNoContent {
				t.Fatalf("disconnecting = %d", status)
			}
		}},
		{"the refresh family is caught replayed", func(t *testing.T, f *oauthFixture, _ *browser, tokens tokenResult) {
			if next := f.refresh(tokens.str("refresh_token")); next.status != http.StatusOK {
				t.Fatalf("the first refresh = %d", next.status)
			}
			// Past the grace, the replaced token presented again is a replay.
			if err := f.s.db.Exec("UPDATE refresh_tokens SET revoked_at = now() - interval '1 minute' WHERE revoked_at IS NOT NULL").Error; err != nil {
				t.Fatal(err)
			}
			if replayed := f.refresh(tokens.str("refresh_token")); replayed.status != http.StatusBadRequest {
				t.Fatalf("the replay = %d, want refused", replayed.status)
			}
		}},
		{"the access token itself is revoked", func(t *testing.T, f *oauthFixture, _ *browser, tokens tokenResult) {
			if revoked := f.token("/oauth2/revoke", f.clientID, f.clientSecret, url.Values{"token": {tokens.str("access_token")}}); revoked.status != http.StatusOK {
				t.Fatalf("revoking = %d", revoked.status)
			}
		}},
		{"an administrator signs the user out everywhere, after the session itself expired", func(t *testing.T, f *oauthFixture, _ *browser, _ tokenResult) {
			// The refresh family outlives the session it began in; signing
			// out reaches it even so.
			if err := f.s.db.Exec("UPDATE user_sessions SET expires_at = now() - interval '1 second'").Error; err != nil {
				t.Fatal(err)
			}
			f.super.must(http.StatusOK, http.MethodDelete, "/users/"+f.userID+"/sessions", nil, nil)
		}},
		{"its refresh token is revoked", func(t *testing.T, f *oauthFixture, _ *browser, tokens tokenResult) {
			if revoked := f.token("/oauth2/revoke", f.clientID, f.clientSecret, url.Values{"token": {tokens.str("refresh_token")}}); revoked.status != http.StatusOK {
				t.Fatalf("revoking = %d", revoked.status)
			}
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			b := f.s.browser()
			verifier, challenge := pkce(t)
			tokens := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
			access := tokens.str("access_token")

			if status := f.userinfoStatus(access); status != http.StatusOK || !f.active(access) {
				t.Fatalf("before: userinfo = %d, active = %v", status, f.active(access))
			}

			c.end(t, f, b, tokens)

			if status := f.userinfoStatus(access); status != http.StatusUnauthorized {
				t.Errorf("userinfo after %s = %d, want 401", c.name, status)
			}
			if f.active(access) {
				t.Errorf("introspection after %s says the access token is active", c.name)
			}
		})
	}
}

// A token another session issued is not ended by this one's sign-out.
func TestLiveASignOutEndsOnlyItsOwnSessionsTokens(t *testing.T) {
	f := newOAuthFixture(t)

	first, second := f.s.browser(), f.s.browser()
	v1, c1 := pkce(t)
	v2, c2 := pkce(t)
	ended := f.exchange(f.signIn(first, c1).Query().Get("code"), v1)
	kept := f.exchange(f.signIn(second, c2).Query().Get("code"), v2)

	first.visit(f.s.root + "/oauth2/logout?id_token_hint=" + url.QueryEscape(ended.str("id_token")))

	if f.active(ended.str("access_token")) {
		t.Error("the signed-out session's token is still active")
	}
	if !f.active(kept.str("access_token")) || f.userinfoStatus(kept.str("access_token")) != http.StatusOK {
		t.Error("another session's token was ended by this sign-out")
	}
}

// An access token always names its audience: an API's, or, with none asked
// for, the client's own (RFC 9068 §3). And it lives ten minutes unless the
// application says otherwise.
func TestLiveAnAccessTokenNamesItsAudience(t *testing.T) {
	f := newOAuthFixture(t)
	verifier, challenge := pkce(t)
	back := f.signInWith(f.s.browser(), challenge, url.Values{"audience": {""}, "scope": {"openid profile"}})
	tokens := f.exchange(back.Query().Get("code"), verifier)
	if tokens.status != http.StatusOK {
		t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
	}

	_, claims := f.verifyJWT(tokens.str("access_token"))
	audience, _ := claims["aud"].([]any)
	names := []string{}
	for _, a := range audience {
		name, _ := a.(string)
		names = append(names, name)
	}
	if !slices.Equal(names, []string{f.clientID}) {
		t.Errorf("aud = %v, want [%s]", claims["aud"], f.clientID)
	}
	if status := f.userinfoStatus(tokens.str("access_token")); status != http.StatusOK {
		t.Errorf("userinfo with it = %d, want 200", status)
	}

	if expires, _ := tokens.body["expires_in"].(float64); expires != 600 {
		t.Errorf("expires_in = %v, want 600 by default", tokens.body["expires_in"])
	}
}
