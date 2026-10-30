package api

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"vard/internal/domain/totp"
)

// A single sign-on connection that links to existing accounts takes over
// whichever accounts its provider names. It may only do so for domains it
// lists and has proved it holds, and the first time it is linked to an
// account that has a second factor, that factor is still asked for.

// oidcConnection makes an OpenID Connect connection to idp and returns its id.
func oidcConnection(t *testing.T, super *client, idp *fakeOIDC, extra map[string]any) string {
	t.Helper()

	body := map[string]any{
		"protocol": "oidc", "name": "Acme", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret, "is_enabled": true,
		"domains": []string{"acme.test"}, "matching": "link",
	}
	for key, value := range extra {
		body[key] = value
	}

	var created struct {
		Connection idOnly `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", body, &created)

	return created.Connection.ID
}

// provedDomain marks a connection's domain as proved, as a DNS check that
// found its record would.
func (s *liveServer) provedDomain(t *testing.T, connection, domain string) {
	t.Helper()

	result := s.db.Exec(`UPDATE sso_domain_verifications SET verified_at = now(), method = 'dns'
		WHERE connection_id = ? AND domain = ?`, connection, domain)
	if result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("marking %s proved: %v (%d rows)", domain, result.Error, result.RowsAffected)
	}
}

func TestLiveALinkingConnectionNeedsItsDomains(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	cases := []struct {
		name string
		body map[string]any
	}{
		{"linking without domains", map[string]any{"domains": []string{}, "matching": "link", "show_on_login": true}},
	}
	for _, c := range cases {
		body := map[string]any{
			"protocol": "oidc", "name": "Partners", "issuer": idp.server.URL,
			"client_id": fakeClientID, "client_secret": fakeClientSecret, "is_enabled": true,
		}
		for key, value := range c.body {
			body[key] = value
		}

		var refused problemBody
		if status := super.do(http.MethodPost, "/sso-connections", body, &refused); status != http.StatusBadRequest || refused.Code != "sso_link_needs_domains" {
			t.Errorf("%s = %d %+v, want 400 sso_link_needs_domains", c.name, status, refused)
		}
	}

	// A connection that lists domains cannot lose them while it links.
	id := oidcConnection(t, super, idp, nil)
	var refused problemBody
	if status := super.do(http.MethodPatch, "/sso-connections/"+id, map[string]any{"domains": []string{}, "show_on_login": true}, &refused); status != http.StatusBadRequest || refused.Code != "sso_link_needs_domains" {
		t.Errorf("removing a linking connection's domains = %d %+v, want 400 sso_link_needs_domains", status, refused)
	}

	// Without a word about matching, a connection without domains does not
	// link: it is made with deny.
	var partners struct {
		Connection struct {
			Matching string `json:"matching"`
		} `json:"connection"`
	}
	super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "oidc", "name": "Others", "issuer": idp.server.URL,
		"client_id": fakeClientID, "client_secret": fakeClientSecret, "is_enabled": true, "show_on_login": true,
	}, &partners)
	if partners.Connection.Matching != "deny" {
		t.Errorf("a connection without domains was made with matching %q, want deny", partners.Connection.Matching)
	}
}

func TestLiveSSOLinksOnlyThroughAProvedDomain(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "grace@acme.test", "first_name": "Grace", "password": "grace-password-1", "confirm_password": "grace-password-1",
		"is_email_verified": true,
	}, nil)
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "lin@elsewhere.test", "first_name": "Lin", "password": "lin-password-1", "confirm_password": "lin-password-1",
		"is_email_verified": true,
	}, nil)
	id := oidcConnection(t, super, idp, nil)

	cases := []struct {
		name   string
		person map[string]any
		want   string
	}{
		{"an account at a domain it has not proved", map[string]any{"sub": "okta|grace", "email": "grace@acme.test"}, "sso_link_refused"},
		{"an account at a domain it does not hold", map[string]any{"sub": "okta|lin", "email": "lin@elsewhere.test"}, "sso_domain_mismatch"},
	}
	for _, c := range cases {
		idp.signs(c.person)
		b := s.browser()
		if reason, _ := idp.signInOIDC(b, "acme").failed(); reason != c.want {
			t.Errorf("%s = %q, want %s", c.name, reason, c.want)
		}
		if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
			t.Errorf("%s signed the browser in", c.name)
		}
	}

	// Somebody new at an unproved domain may still be made an account: that
	// takes nobody's over.
	idp.signs(map[string]any{"sub": "okta|new", "email": "new@acme.test", "email_verified": true})
	if reason, failed := idp.signInOIDC(s.browser(), "acme").failed(); failed {
		t.Errorf("a new person at an unproved domain = %q, want an account made", reason)
	}

	s.provedDomain(t, id, "acme.test")
	idp.signs(map[string]any{"sub": "okta|grace", "email": "grace@acme.test"})
	b := s.browser()
	if reason, failed := idp.signInOIDC(b, "acme").failed(); failed {
		t.Fatalf("linking at a proved domain = %q", reason)
	}
	if email, _, _ := b.me(); email != "grace@acme.test" {
		t.Errorf("linked to %q, want grace@acme.test", email)
	}
}

func TestLiveTheFirstSSOLinkAsksForTheAccountsSecondFactor(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()
	idp := newFakeOIDC(t)

	const email = "ada@acme.test"
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": email, "first_name": "Ada", "password": adaPassword, "confirm_password": adaPassword,
		"is_email_verified": true,
	}, nil)
	super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, super), map[string]any{
		"steps": []string{"identifier", "password", "totp"},
	}, nil)

	owner := s.browser()
	if status := owner.account(http.MethodPost, "/login", map[string]string{"email": email, "password": adaPassword}, nil); status != http.StatusOK {
		t.Fatalf("Ada's own sign-in = %d", status)
	}
	secret, _ := addAuthenticator(t, owner)

	id := oidcConnection(t, super, idp, nil)
	s.provedDomain(t, id, "acme.test")
	idp.signs(map[string]any{"sub": "okta|ada", "email": email})

	// The first link waits for Ada's authenticator, and without it there is
	// no session and no link.
	b := s.browser()
	result := idp.signInOIDC(b, "acme")
	if !strings.HasPrefix(result.landed.String(), testAccountURL+"/second-factor") {
		t.Fatalf("the first link landed at %s, want the second-factor page", result.landed)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Fatal("the first link signed in before the second factor")
	}
	var links int64
	if err := s.db.Table("sso_identities").Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links != 0 {
		t.Fatalf("%d identities were linked before the second factor", links)
	}

	// A stranger who stops there has linked nothing: the next try asks again.
	if again := idp.signInOIDC(s.browser(), "acme"); !strings.HasPrefix(again.landed.String(), testAccountURL+"/second-factor") {
		t.Fatalf("a second try after an abandoned link landed at %s, want the second-factor page again", again.landed)
	}

	status, _ := answerSecondFactor(b, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))})
	if status != http.StatusOK {
		t.Fatalf("Ada's code = %d", status)
	}
	if got, _, _ := b.me(); got != email {
		t.Fatalf("signed in as %q after the code, want %s", got, email)
	}
	if err := s.db.Table("sso_identities").Count(&links).Error; err != nil {
		t.Fatal(err)
	}
	if links != 1 {
		t.Errorf("%d identities linked after the second factor, want 1", links)
	}

	// Linked, the provider answers for its people's second factor.
	later := s.browser()
	if reason, failed := idp.signInOIDC(later, "acme").failed(); failed {
		t.Fatalf("a later sign-in through the link = %q", reason)
	}
	if got, _, _ := later.me(); got != email {
		t.Errorf("a later sign-in through the link = %q, want %s signed in at once", got, email)
	}
}
