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

// A session ended on purpose — by its owner from another browser, or by an
// administrator — takes the refresh families issued in it along: the device
// it was on, a lost laptop, is shut out of its applications, not only of the
// sign-in pages. An application's own logout leaves its offline_access be,
// which outlives the browser's session (OpenID Connect Core §11).
func TestLiveEndingASessionEndsItsRefreshFamilies(t *testing.T) {
	cases := []struct {
		name         string
		end          func(t *testing.T, f *oauthFixture, laptop *browser, tokens tokenResult)
		refreshEnded bool
	}{
		{"its owner ends it from another browser", func(t *testing.T, f *oauthFixture, _ *browser, _ tokenResult) {
			phone := f.s.browser()
			_, challenge := pkce(t)
			f.signIn(phone, challenge)
			if status := phone.account(http.MethodDelete, "/sessions/"+otherSession(t, phone), nil, nil); status != http.StatusNoContent {
				t.Fatalf("ending the laptop's session = %d", status)
			}
		}, true},
		{"an administrator ends it", func(t *testing.T, f *oauthFixture, laptop *browser, _ tokenResult) {
			f.super.must(http.StatusNoContent, http.MethodDelete, "/user-sessions/"+currentSession(t, laptop), nil, nil)
		}, true},
		{"the application signs the browser out", func(t *testing.T, f *oauthFixture, laptop *browser, tokens tokenResult) {
			laptop.visit(f.s.root + "/oauth2/logout?id_token_hint=" + url.QueryEscape(tokens.str("id_token")))
		}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			laptop := f.s.browser()
			verifier, challenge := pkce(t)
			tokens := f.exchange(f.signIn(laptop, challenge).Query().Get("code"), verifier)
			refresh := tokens.str("refresh_token")
			if tokens.status != http.StatusOK || refresh == "" {
				t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
			}

			c.end(t, f, laptop, tokens)

			if status := laptop.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
				t.Fatalf("the ended session's /me = %d, want 401", status)
			}

			answer := f.refresh(refresh)
			switch {
			case c.refreshEnded && (answer.status != http.StatusBadRequest || answer.str("error") != "invalid_grant"):
				t.Errorf("refresh after %s = %d %v, want invalid_grant", c.name, answer.status, answer.body)
			case c.refreshEnded && f.active(refresh):
				t.Errorf("introspection after %s says the refresh token is active", c.name)
			case !c.refreshEnded && answer.status != http.StatusOK:
				t.Errorf("refresh after %s = %d %v, want offline access kept", c.name, answer.status, answer.body)
			}
		})
	}
}

// currentSession is the id of the session the browser is signed in with.
func currentSession(t *testing.T, b *browser) string {
	t.Helper()
	for _, s := range sessionsOf(t, b) {
		if s.Current {
			return s.ID
		}
	}
	t.Fatal("the browser has no current session")
	return ""
}

// otherSession is the id of the one session of the user's that is not the
// browser's own.
func otherSession(t *testing.T, b *browser) string {
	t.Helper()
	for _, s := range sessionsOf(t, b) {
		if !s.Current {
			return s.ID
		}
	}
	t.Fatal("the user has no other session")
	return ""
}

type listedSession struct {
	ID      string `json:"id"`
	Current bool   `json:"current"`
}

func sessionsOf(t *testing.T, b *browser) []listedSession {
	t.Helper()
	var list struct {
		Sessions []listedSession `json:"sessions"`
	}
	if status := b.account(http.MethodGet, "/sessions", nil, &list); status != http.StatusOK {
		t.Fatalf("listing sessions = %d", status)
	}
	return list.Sessions
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

// Whatever ends a user's access ends every token they were issued, at once
// and in the database: refresh families, and access tokens through the
// user-wide revocation — those of a session that had already expired
// included, which ending sessions alone would miss. Turning a user off and
// on again brings none of it back.
func TestLiveEverythingThatEndsAUsersAccessEndsTheirTokens(t *testing.T) {
	cases := []struct {
		name string
		end  func(t *testing.T, f *oauthFixture, b *browser)
	}{
		{"they change their password", func(t *testing.T, f *oauthFixture, b *browser) {
			if status := b.account(http.MethodPost, "/password", map[string]string{"current_password": adaPassword, "new_password": "ada-new-password-1"}, nil); status != http.StatusNoContent && status != http.StatusOK {
				t.Fatalf("changing the password = %d", status)
			}
		}},
		{"they reset their password", func(t *testing.T, f *oauthFixture, _ *browser) {
			f.s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": adaEmail}, nil)
			token := tokenIn(t, f.s.mail.waitFor(t, adaEmail, "/reset-password").Body, "/reset-password")
			if status := f.s.browser().account(http.MethodPost, "/reset-password", map[string]string{"token": token, "password": "ada-new-password-1"}, nil); status != http.StatusOK {
				t.Fatalf("resetting = %d", status)
			}
		}},
		{"an administrator sets a new password", func(t *testing.T, f *oauthFixture, _ *browser) {
			f.super.must(http.StatusOK, http.MethodPatch, "/users/"+f.userID, map[string]any{
				"email": adaEmail, "password": "ada-new-password-1", "confirm_password": "ada-new-password-1",
			}, nil)
		}},
		{"an administrator turns them off", func(t *testing.T, f *oauthFixture, _ *browser) {
			f.super.must(http.StatusOK, http.MethodPatch, "/users/"+f.userID, map[string]any{
				"email": adaEmail, "first_name": "Ada", "last_name": "Lovelace", "is_active": false,
			}, nil)
			// On again: nothing issued before comes back with them.
			f.super.must(http.StatusOK, http.MethodPatch, "/users/"+f.userID, map[string]any{
				"email": adaEmail, "first_name": "Ada", "last_name": "Lovelace", "is_active": true,
			}, nil)
		}},
		{"an administrator signs them out everywhere", func(t *testing.T, f *oauthFixture, _ *browser) {
			f.super.must(http.StatusOK, http.MethodDelete, "/users/"+f.userID+"/sessions", nil, nil)
		}},
		{"an administrator deletes them", func(t *testing.T, f *oauthFixture, _ *browser) {
			f.super.must(http.StatusNoContent, http.MethodDelete, "/users/"+f.userID, nil, nil)
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newOAuthFixture(t)

			// The tokens come from a browser whose session has since expired,
			// so ending sessions does not reach them; only the user-wide
			// revocation does. The person acts from another, live browser.
			old := f.s.browser()
			verifier, challenge := pkce(t)
			tokens := f.exchange(f.signIn(old, challenge).Query().Get("code"), verifier)
			access, refresh := tokens.str("access_token"), tokens.str("refresh_token")
			if tokens.status != http.StatusOK || refresh == "" {
				t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
			}
			if err := f.s.db.Exec("UPDATE user_sessions SET expires_at = now() - interval '1 second' WHERE id = ?", currentSession(t, old)).Error; err != nil {
				t.Fatal(err)
			}

			b := f.s.browser()
			_, again := pkce(t)
			f.signIn(b, again)
			if status := f.userinfoStatus(access); status != http.StatusOK {
				t.Fatalf("before: userinfo = %d", status)
			}

			c.end(t, f, b)

			if status := f.userinfoStatus(access); status != http.StatusUnauthorized {
				t.Errorf("userinfo after %s = %d, want 401", c.name, status)
			}
			if f.active(access) || f.active(refresh) {
				t.Errorf("introspection after %s says a token is active", c.name)
			}
			if again := f.refresh(refresh); again.str("error") != "invalid_grant" {
				t.Errorf("refresh after %s = %d %v, want invalid_grant", c.name, again.status, again.body)
			}

			var rows int64
			if err := f.s.db.Raw("SELECT count(*) FROM token_revocations WHERE key = ?", "user:"+f.userID).Scan(&rows).Error; err != nil {
				t.Fatal(err)
			}
			if rows != 1 {
				t.Errorf("after %s the user-wide revocation is %d rows, want 1", c.name, rows)
			}
		})
	}
}
