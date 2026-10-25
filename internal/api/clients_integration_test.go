package api

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"vard/internal/model"
)

// These tests hold the provider to what client libraries rely on: PKCE for
// every client without credentials, a refresh that survives a lost answer,
// and a native app's redirect on whatever loopback port it opened.

// agoRefreshRotations moves every replaced refresh token's replacement back
// by d, as if that long had passed since.
func (s *liveServer) agoRefreshRotations(d time.Duration) {
	s.t.Helper()

	if err := s.db.Exec("UPDATE refresh_tokens SET revoked_at = revoked_at - make_interval(secs => ?) WHERE revoked_at IS NOT NULL", d.Seconds()+1).Error; err != nil {
		s.t.Fatal(err)
	}
}

// liveRefreshTokens is how many of a user's refresh tokens are live: one per
// family, whatever retries there were.
func (s *liveServer) liveRefreshTokens(userID string) int64 {
	s.t.Helper()

	var count int64
	if err := s.db.Model(&model.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", userID).Count(&count).Error; err != nil {
		s.t.Fatal(err)
	}
	return count
}

// changeApplication changes an application in the store, past the panel's
// rules: for a row the panel could not make, as an older version or a
// hand-edited database might have left it.
func (s *liveServer) changeApplication(clientID string, change func(*model.Application)) {
	s.t.Helper()

	ctx := context.Background()
	app, err := s.store.ApplicationByClientID(ctx, clientID)
	if err != nil {
		s.t.Fatal(err)
	}
	change(app)
	if err := s.store.SaveApplication(ctx, app); err != nil {
		s.t.Fatal(err)
	}
}

func (f *oauthFixture) refresh(token string) tokenResult {
	f.s.t.Helper()
	return f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {token},
	})
}

// A phone sends a refresh, loses the answer, and retries with the token it
// still has. Within the grace that is a retry, not theft: it gets the very
// refresh token the lost answer carried, so the family still has one live
// token. Once the grace is over the sealed copy is swept away, and the same
// token is a replay that takes the family.
func TestLiveRefreshSurvivesALostAnswer(t *testing.T) {
	f := newOAuthFixture(t)
	verifier, challenge := pkce(t)
	tokens := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier)
	first := tokens.str("refresh_token")
	if tokens.status != http.StatusOK || first == "" {
		t.Fatalf("exchange = %d %v", tokens.status, tokens.body)
	}

	lost := f.refresh(first)
	if lost.status != http.StatusOK {
		t.Fatalf("first refresh = %d %v", lost.status, lost.body)
	}

	retried := f.refresh(first)
	if retried.status != http.StatusOK || retried.str("refresh_token") == "" || retried.str("access_token") == "" {
		t.Fatalf("the retry within the grace = %d %v, want new tokens", retried.status, retried.body)
	}
	if retried.str("refresh_token") != lost.str("refresh_token") {
		t.Error("the retry was given a new refresh token, want the one the lost answer carried")
	}
	if retried.str("access_token") == lost.str("access_token") {
		t.Error("the retry was given the lost access token, want a new one")
	}
	if live := f.s.liveRefreshTokens(f.userID); live != 1 {
		t.Errorf("%d live refresh tokens after the retry, want 1: the lost one is replaced, not kept beside it", live)
	}

	// The retry's token goes on as any other.
	next := f.refresh(retried.str("refresh_token"))
	if next.status != http.StatusOK {
		t.Fatalf("refresh with the retry's token = %d %v", next.status, next.body)
	}

	// Past the grace, the sealed copies go, and the first token presented
	// again is a replay.
	f.s.agoRefreshRotations(model.RefreshReuseGrace)
	if _, err := f.s.store.Sweep(t.Context(), time.Now(), time.Time{}); err != nil {
		t.Fatal(err)
	}
	var sealed int64
	if err := f.s.db.Model(&model.RefreshToken{}).Where("successor_sealed IS NOT NULL").Count(&sealed).Error; err != nil {
		t.Fatal(err)
	}
	if sealed != 0 {
		t.Errorf("%d sealed refresh tokens kept past the grace, want none", sealed)
	}
	if replay := f.refresh(first); replay.status != http.StatusBadRequest || replay.str("error") != "invalid_grant" {
		t.Errorf("the first token after the grace = %d %v, want invalid_grant", replay.status, replay.body)
	}
	if after := f.refresh(next.str("refresh_token")); after.status != http.StatusBadRequest {
		t.Errorf("the family's live token after the replay = %d %v, want it revoked with the family", after.status, after.body)
	}
}

// Two requests refreshing with one token at once — two tabs, or a retry that
// overtakes the first — both get tokens, and still only one is left live.
func TestLiveRefreshTakesTwoAtOnce(t *testing.T) {
	f := newOAuthFixture(t)
	verifier, challenge := pkce(t)
	first := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier).str("refresh_token")

	results := make([]tokenResult, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Go(func() { results[i] = f.refresh(first) })
	}
	wg.Wait()

	for i, result := range results {
		if result.status != http.StatusOK {
			t.Errorf("concurrent refresh %d = %d %v, want tokens", i, result.status, result.body)
		}
	}
	if live := f.s.liveRefreshTokens(f.userID); live != 1 {
		t.Errorf("%d live refresh tokens, want 1", live)
	}
}

// Two tabs refresh with one token a moment apart. Each keeps what it was
// given, and refreshes again only when its access token runs out, long after
// the grace. Neither may be signed out for it: within the grace the second
// is given the very token the first was, not a new one that leaves the
// first holding a dead token to be caught as reuse later.
func TestLiveRefreshTwoTabsKeepTheirTokens(t *testing.T) {
	f := newOAuthFixture(t)
	verifier, challenge := pkce(t)
	t1 := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier).str("refresh_token")

	tabA := f.refresh(t1)
	tabB := f.refresh(t1)
	if tabA.status != http.StatusOK || tabB.status != http.StatusOK {
		t.Fatalf("the two refreshes = %d %v and %d %v", tabA.status, tabA.body, tabB.status, tabB.body)
	}

	// Long after the grace, tab A refreshes with what it was given.
	f.s.agoRefreshRotations(model.RefreshReuseGrace)
	later := f.refresh(tabA.str("refresh_token"))
	if later.status != http.StatusOK {
		t.Fatalf("tab A's refresh after the grace = %d %v, want tokens: it was signed out for tab B's refresh", later.status, later.body)
	}
	if live := f.s.liveRefreshTokens(f.userID); live != 1 {
		t.Errorf("%d live refresh tokens, want 1", live)
	}
}

// The grace is for a lost answer, not for a family that has ended: a token
// replaced just before its family was revoked stays dead.
func TestLiveRefreshGraceEndsWithTheFamily(t *testing.T) {
	f := newOAuthFixture(t)
	verifier, challenge := pkce(t)
	first := f.exchange(f.signIn(f.s.browser(), challenge).Query().Get("code"), verifier).str("refresh_token")

	second := f.refresh(first).str("refresh_token")
	if revoked := f.token("/oauth2/revoke", f.clientID, f.clientSecret, url.Values{"token": {second}}); revoked.status != http.StatusOK {
		t.Fatalf("revoke = %d %v", revoked.status, revoked.body)
	}

	if retry := f.refresh(first); retry.status != http.StatusBadRequest || retry.str("error") != "invalid_grant" {
		t.Errorf("a replaced token after its family was revoked = %d %v, want invalid_grant", retry.status, retry.body)
	}
	if live := f.s.liveRefreshTokens(f.userID); live != 0 {
		t.Errorf("%d live refresh tokens after revocation, want none", live)
	}
}

// A public client has nothing but PKCE to tie a code to the app that asked
// for it, so it always uses PKCE — even if its row says otherwise. And a
// code asked for with a challenge is never exchanged without its verifier,
// whatever the application requires (the downgrade RFC 9700 §2.1.1 warns of).
func TestLivePKCECannotBeLeftOut(t *testing.T) {
	f := newOAuthFixture(t)

	spaID, _, _ := f.register(map[string]any{
		"name": "Storefront", "type": "spa",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"https://store.example.com/callback"},
		"scopes":        []string{"openid"},
	})
	f.s.changeApplication(spaID, func(app *model.Application) { app.RequirePKCE = false })

	location := f.s.browser().visit(f.s.root + "/oauth2/authorize?" + url.Values{
		"response_type": {"code"}, "client_id": {spaID}, "scope": {"openid"},
		"redirect_uri": {"https://store.example.com/callback"}, "state": {"s"},
	}.Encode())
	if location.Query().Get("error") != "invalid_request" || !strings.Contains(location.Query().Get("error_description"), "code_challenge") {
		t.Errorf("a public client's request without PKCE went to %s, want invalid_request about code_challenge", location)
	}

	// A confidential client that does not require PKCE, but asked with a
	// challenge, still has to prove the verifier.
	f.s.changeApplication(f.clientID, func(app *model.Application) { app.RequirePKCE = false })
	verifier, challenge := pkce(t)
	code := f.signIn(f.s.browser(), challenge).Query().Get("code")

	if downgraded := f.exchange(code, ""); downgraded.status != http.StatusBadRequest || downgraded.str("error") != "invalid_grant" {
		t.Errorf("exchange without the verifier = %d %v, want invalid_grant", downgraded.status, downgraded.body)
	}
	// The failed attempt spent the code, as any wrong answer does.
	if spent := f.exchange(code, verifier); spent.status != http.StatusBadRequest {
		t.Errorf("the code after a failed exchange = %d %v, want it spent", spent.status, spent.body)
	}
}

// A native app opens whichever loopback port is free and redirects there: it
// registers the loopback URI without a port and is answered on any (RFC 8252
// §7.3), but the path and everything else still match exactly.
func TestLiveNativeAppRedirectsToAnyLoopbackPort(t *testing.T) {
	f := newOAuthFixture(t)

	nativeID, _, _ := f.register(map[string]any{
		"name": "Desktop", "type": "native",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{"http://127.0.0.1/callback", "com.example.desktop:/callback"},
		"scopes":        []string{"openid"},
	})

	signIn := func(redirect string) (*url.URL, string) {
		t.Helper()
		verifier, challenge := pkce(t)
		b := f.s.browser()
		login := b.visit(f.s.root + "/oauth2/authorize?" + url.Values{
			"response_type": {"code"}, "client_id": {nativeID}, "scope": {"openid"}, "redirect_uri": {redirect},
			"state": {"s"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"},
		}.Encode())
		if !strings.HasPrefix(login.String(), testAccountURL+"/login?") {
			return login, verifier
		}

		var out struct {
			RedirectTo string `json:"redirect_to"`
		}
		status := b.account(http.MethodPost, "/login", map[string]string{"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword}, &out)
		if status != http.StatusOK {
			t.Fatalf("login = %d", status)
		}
		back, err := url.Parse(out.RedirectTo)
		if err != nil {
			t.Fatal(err)
		}
		return back, verifier
	}

	for _, redirect := range []string{"http://127.0.0.1:53111/callback", "http://127.0.0.1:61999/callback", "com.example.desktop:/callback"} {
		t.Run(redirect, func(t *testing.T) {
			back, verifier := signIn(redirect)
			if !strings.HasPrefix(back.String(), redirect+"?") || back.Query().Get("code") == "" {
				t.Fatalf("signed in back to %s, want %s with a code", back, redirect)
			}

			tokens := f.token("/oauth2/token", nativeID, "", url.Values{
				"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
				"redirect_uri": {redirect}, "code_verifier": {verifier},
			})
			if tokens.status != http.StatusOK || tokens.str("id_token") == "" {
				t.Errorf("exchange = %d %v", tokens.status, tokens.body)
			}
		})
	}

	for _, redirect := range []string{"http://127.0.0.1:53111/other", "http://localhost:53111/callback", "http://127.0.0.1:53111/callback?x=1"} {
		t.Run("refused: "+redirect, func(t *testing.T) {
			back, _ := signIn(redirect)
			if !strings.HasPrefix(back.String(), testAccountURL+"/error?") {
				t.Errorf("an unregistered %s went to %s, want the error page", redirect, back)
			}
		})
	}
}
