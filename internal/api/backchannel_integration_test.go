package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// receiver is an application's back-channel logout endpoint: it keeps every
// logout token posted to it, and answers with what `answers` says, in turn,
// then 200.
type receiver struct {
	url     string
	tokens  chan string
	answers []int
}

func newReceiver(t *testing.T, answers ...int) *receiver {
	t.Helper()

	r := &receiver{tokens: make(chan string, 16), answers: answers}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost || req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("the logout came as %s %s", req.Method, req.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(req.Body)
		form, _ := url.ParseQuery(string(body))
		r.tokens <- form.Get("logout_token")
		status := http.StatusOK
		if len(r.answers) > 0 {
			status, r.answers = r.answers[0], r.answers[1:]
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	r.url = server.URL + "/backchannel-logout"

	return r
}

func (r *receiver) wait(t *testing.T) string {
	t.Helper()

	select {
	case token := <-r.tokens:
		return token
	case <-time.After(5 * time.Second):
		t.Fatal("no logout token was posted")
		return ""
	}
}

func (r *receiver) none(t *testing.T) {
	t.Helper()

	select {
	case token := <-r.tokens:
		t.Fatalf("a logout token was posted when none should be: %s", token)
	case <-time.After(300 * time.Millisecond):
	}
}

// An application with a back-channel logout URI is told, server to server,
// whenever a session it signed the user in through ends — however it ends —
// with a logout token naming the user and the session from its ID token.
func TestLiveBackchannelLogout(t *testing.T) {
	tests := []struct {
		name string
		end  func(f *oauthFixture, b *browser, idToken string)
	}{
		{
			name: "the application signs the user out",
			end: func(f *oauthFixture, b *browser, idToken string) {
				b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {idToken}}.Encode())
			},
		},
		{
			name: "an administrator signs the user out everywhere",
			end: func(f *oauthFixture, _ *browser, _ string) {
				f.super.must(http.StatusOK, http.MethodDelete, "/users/"+f.userID+"/sessions", nil, nil)
			},
		},
		{
			name: "an administrator deletes the user",
			end: func(f *oauthFixture, _ *browser, _ string) {
				f.super.must(http.StatusNoContent, http.MethodDelete, "/users/"+f.userID, nil, nil)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			told := newReceiver(t)

			// The fixture's own application has no URI and must not be told.
			f.clientID, f.clientSecret, f.appID = f.register(map[string]any{
				"name": "Back office", "type": "web",
				"grant_types":            []string{"authorization_code"},
				"redirect_uris":          []string{shopRedirect},
				"scopes":                 []string{"openid", "email"},
				"backchannel_logout_uri": told.url,
			})

			b := f.s.browser()
			verifier, challenge := pkce(t)
			back := f.signInWith(b, challenge, url.Values{"scope": {"openid email"}, "audience": {""}})
			tokens := f.exchange(back.Query().Get("code"), verifier)
			if tokens.str("id_token") == "" {
				t.Fatalf("no ID token: %d %v", tokens.status, tokens.body)
			}
			_, id := f.verifyJWT(tokens.str("id_token"))
			if id["sid"] == nil {
				t.Fatalf("the ID token has no sid: %v", id)
			}

			test.end(f, b, tokens.str("id_token"))

			header, claims := f.verifyJWT(told.wait(t))
			if header["typ"] != "logout+jwt" {
				t.Errorf("typ = %v, want logout+jwt", header["typ"])
			}
			want := map[string]any{"iss": f.s.cfg.Issuer, "aud": f.clientID, "sub": f.userID, "sid": id["sid"]}
			for claim, value := range want {
				if claims[claim] != value {
					t.Errorf("%s = %v, want %v", claim, claims[claim], value)
				}
			}
			events, _ := claims["events"].(map[string]any)
			if _, ok := events["http://schemas.openid.net/event/backchannel-logout"]; !ok {
				t.Errorf("events = %v, want the back-channel logout event", claims["events"])
			}
			if _, ok := claims["nonce"]; ok {
				t.Error("a logout token must not carry a nonce")
			}
			if claims["jti"] == nil || claims["iat"] == nil || claims["exp"] == nil {
				t.Errorf("jti, iat and exp are required: %v", claims)
			}

			told.none(t)
		})
	}
}

// Only applications that were given an ID token in the session are told: one
// with a URI that the user never signed in to hears nothing.
func TestLiveBackchannelLogoutTellsOnlyTheSessionsApplications(t *testing.T) {
	f := newOAuthFixture(t)
	told := newReceiver(t)

	f.register(map[string]any{
		"name": "Never used", "type": "web",
		"grant_types":            []string{"authorization_code"},
		"redirect_uris":          []string{shopRedirect},
		"scopes":                 []string{"openid"},
		"backchannel_logout_uri": told.url,
	})

	b := f.s.browser()
	verifier, challenge := pkce(t)
	tokens := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {tokens.str("id_token")}}.Encode())

	told.none(t)
}

// A refresh keeps the session: the new ID token names the sid the first one
// did, so the application can match the logout token to it.
func TestLiveRefreshedIDTokenKeepsTheSid(t *testing.T) {
	f := newOAuthFixture(t)
	b := f.s.browser()

	verifier, challenge := pkce(t)
	first := f.exchange(f.signIn(b, challenge).Query().Get("code"), verifier)
	_, before := f.verifyJWT(first.str("id_token"))

	refreshed := f.token("/oauth2/token", f.clientID, f.clientSecret, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {first.str("refresh_token")},
	})
	if refreshed.str("id_token") == "" {
		t.Fatalf("the refresh gave no ID token: %d %v", refreshed.status, refreshed.body)
	}
	_, after := f.verifyJWT(refreshed.str("id_token"))

	if before["sid"] == nil || after["sid"] != before["sid"] {
		t.Errorf("sid went from %v to %v, want it kept", before["sid"], after["sid"])
	}
}

// A logout the application could not take is tried again; one it refuses is
// not, and ends up in the activity log, where administrators see which
// application missed it.
func TestLiveBackchannelLogoutIsTriedAgainAndLogged(t *testing.T) {
	tests := []struct {
		name    string
		answers []int
		posts   int
		logged  bool
	}{
		{name: "an application that fails once gets it on the next try", answers: []int{http.StatusServiceUnavailable}, posts: 2},
		{name: "an application that refuses it is not asked again", answers: []int{http.StatusBadRequest}, posts: 1, logged: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			told := newReceiver(t, test.answers...)
			f.clientID, f.clientSecret, f.appID = f.register(map[string]any{
				"name": "Back office", "type": "web",
				"grant_types":            []string{"authorization_code"},
				"redirect_uris":          []string{shopRedirect},
				"scopes":                 []string{"openid", "email"},
				"backchannel_logout_uri": told.url,
			})

			b := f.s.browser()
			verifier, challenge := pkce(t)
			back := f.signInWith(b, challenge, url.Values{"scope": {"openid email"}, "audience": {""}})
			tokens := f.exchange(back.Query().Get("code"), verifier)
			b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {tokens.str("id_token")}}.Encode())

			for range test.posts {
				told.wait(t)
			}
			told.none(t)

			logged := func() bool {
				var page struct {
					Logs []struct {
						Action string `json:"action"`
					} `json:"logs"`
				}
				f.super.must(http.StatusOK, http.MethodGet, "/logs?limit=50", nil, &page)
				for _, entry := range page.Logs {
					if entry.Action == "application.logout_undelivered" {
						return true
					}
				}
				return false
			}
			if got := logged(); got != test.logged {
				t.Errorf("the activity log has the undelivered logout: %v, want %v", got, test.logged)
			}
		})
	}
}

// An application that ends its session in the browser is loaded in a hidden
// frame when the browser signs out, with iss and the sid its ID token
// carried; the page's policy frames that application and nothing else, and
// it is shown once.
func TestLiveFrontchannelLogout(t *testing.T) {
	f := newOAuthFixture(t)
	const frontchannel = "https://shop.example.com/frontchannel-logout?from=idp"
	f.clientID, f.clientSecret, f.appID = f.register(map[string]any{
		"name": "Single page", "type": "web",
		"grant_types":             []string{"authorization_code"},
		"redirect_uris":           []string{shopRedirect},
		"scopes":                  []string{"openid", "email"},
		"frontchannel_logout_uri": frontchannel,
	})

	b := f.s.browser()
	verifier, challenge := pkce(t)
	back := f.signInWith(b, challenge, url.Values{"scope": {"openid email"}, "audience": {""}})
	tokens := f.exchange(back.Query().Get("code"), verifier)
	_, id := f.verifyJWT(tokens.str("id_token"))

	at := b.visit(f.s.root + "/oauth2/logout?" + url.Values{"id_token_hint": {tokens.str("id_token")}}.Encode())
	if !strings.HasPrefix(at.String(), f.s.cfg.Issuer+"/oauth2/logout/frontchannel?") {
		t.Fatalf("the logout led to %s, want the front-channel page", at)
	}

	show := func() *http.Response {
		res, err := b.http.Get(at.String())
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	res := show()
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("the page = %d", res.StatusCode)
	}
	policy := res.Header.Get("Content-Security-Policy")
	if !strings.Contains(policy, "frame-src https://shop.example.com;") || strings.Contains(policy, "unsafe") {
		t.Errorf("the page's policy is %q, want it to frame the application and nothing else", policy)
	}
	want := url.Values{"iss": {f.s.cfg.Issuer}, "sid": {id["sid"].(string)}}.Encode()
	if !strings.Contains(string(page), "frontchannel-logout?from=idp&amp;"+strings.ReplaceAll(want, "&", "&amp;")) {
		t.Errorf("the page does not frame %s with %s:\n%s", frontchannel, want, page)
	}

	again := show()
	again.Body.Close()
	if again.StatusCode != http.StatusFound || !strings.HasPrefix(again.Header.Get("Location"), testAccountURL+"/logged-out") {
		t.Errorf("the page a second time = %d %s, want the signed-out page", again.StatusCode, again.Header.Get("Location"))
	}
}
