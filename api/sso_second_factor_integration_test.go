package api

import (
	"encoding/base64"
	"html"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	dsig "github.com/russellhaering/goxmldsig"

	"vard/internal/domain/totp"
	"vard/internal/interfaces/api/kit/session"
)

// A single sign-on to an application whose flow requires a second factor:
// the account's own (local), or the provider's when the connection trusts it
// and the provider's signed answer shows one — falling back to the account's
// own when it does not.

const (
	samlMFA      = "urn:example:ac:classes:mfa"
	samlPassword = "urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport"
)

// ssoInto walks b through an OpenID Connect connection for an authorization
// request, and returns where it landed.
func (f *fakeOIDC) ssoInto(b *browser, slug, request string) *url.URL {
	f.t.Helper()

	atProvider := b.visit(b.s.root + "/oauth2/sso/" + slug + "/start?request=" + url.QueryEscape(request))
	callback := b.visit(atProvider.String())
	return b.visit(callback.String())
}

// samlForm walks b to the provider for an authorization request and returns
// the form its page would post back.
func (f *fakeSAML) samlForm(t *testing.T, b *browser, slug, request string) url.Values {
	t.Helper()

	atProvider := b.visit(b.s.root + "/oauth2/sso/" + slug + "/start?request=" + url.QueryEscape(request))
	res, err := http.Get(atProvider.String())
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()

	form := url.Values{}
	for _, match := range formField.FindAllStringSubmatch(string(body), -1) {
		form.Set(match[1], html.UnescapeString(match[2]))
	}
	if form.Get("SAMLResponse") == "" {
		t.Fatalf("the provider answered %d without a response: %.300s", res.StatusCode, body)
	}
	return form
}

// classMaker has crewjam's provider say the sign-in was of one class.
type classMaker struct{ class string }

func (m classMaker) MakeAssertion(req *saml.IdpAuthnRequest, session *saml.Session) error {
	if err := (saml.DefaultAssertionMaker{}).MakeAssertion(req, session); err != nil {
		return err
	}
	for i := range req.Assertion.AuthnStatements {
		req.Assertion.AuthnStatements[i].AuthnContext.AuthnContextClassRef = &saml.AuthnContextClassRef{Value: m.class}
	}
	return nil
}

// resigned is the provider's response with change made to it outside the
// signed assertion, signed again as a whole by the provider's own key — a
// response the provider vouches for, whose assertion is untouched.
func (f *fakeSAML) resigned(t *testing.T, form url.Values, change func(response *etree.Element)) url.Values {
	t.Helper()

	raw, err := base64.StdEncoding.DecodeString(form.Get("SAMLResponse"))
	if err != nil {
		t.Fatal(err)
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(raw); err != nil {
		t.Fatal(err)
	}
	response := doc.Root()
	// The provider's signature covered what it sent, not this.
	for _, signature := range response.SelectElements("Signature") {
		response.RemoveChild(signature)
	}
	change(response)

	signer, err := dsig.NewSigningContext(f.idp.Signer, [][]byte{f.idp.Certificate.Raw})
	if err != nil {
		t.Fatal(err)
	}
	if err := signer.SetSignatureMethod(dsig.RSASHA256SignatureMethod); err != nil {
		t.Fatal(err)
	}
	signer.Canonicalizer = dsig.MakeC14N10ExclusiveCanonicalizerWithPrefixList("")
	signed, err := signer.SignEnveloped(response)
	if err != nil {
		t.Fatal(err)
	}
	doc.SetRoot(signed)

	out, err := doc.WriteToBytes()
	if err != nil {
		t.Fatal(err)
	}
	again := url.Values{}
	for key, values := range form {
		again[key] = values
	}
	again.Set("SAMLResponse", base64.StdEncoding.EncodeToString(out))
	return again
}

// fellBack reports whether a sign-in through a connection landed on the
// second-factor page, as one falling back to the account's own factor does.
func fellBack(at *url.URL) bool {
	return strings.HasPrefix(at.String(), testAccountURL+"/second-factor")
}

// The connection's own word decides nothing in local mode: a requiring flow
// holds the sign-in for the account's own factor (or its setup), whatever
// the provider says it proved.
func TestLiveALocalConnectionHoldsForTheAccountsOwnFactor(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)
	idp := newFakeOIDC(t)
	oidcConnection(t, f.super, idp, map[string]any{"create_users": true})
	idp.signs(map[string]any{"sub": "okta|grace", "email": "grace@acme.test", "email_verified": true, "amr": []string{"pwd", "mfa"}})

	b := f.s.browser()
	verifier, challenge := pkce(t)
	landed := idp.ssoInto(b, "acme", page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"))
	if !fellBack(landed) {
		t.Fatalf("a local connection's sign-in went to %s, want the second-factor page", landed)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("a session was opened before the account's own factor")
	}

	// Grace has none yet: she sets one up, and only then reaches the vault.
	var offer enrolOffer
	b.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if !offer.SecondFactor.Enrol {
		t.Fatalf("the second-factor page = %+v, want setup", offer.SecondFactor)
	}
	secret := startAuthenticator(t, b)
	var done enrolled
	if status := b.account(http.MethodPost, "/login/second-factor/authenticator/confirm",
		map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}, &done); status != http.StatusOK {
		t.Fatalf("setting up the factor = %d %s", status, done.Code)
	}
	back, _ := url.Parse(done.RedirectTo)
	vault.exchange(codeBack(t, back, vaultRedirect), verifier)

	// The next sign-in through the connection asks for that factor.
	again := f.s.browser()
	_, challenge = pkce(t)
	if landed := idp.ssoInto(again, "acme", page(t, again.visit(vault.authorizeURL(challenge, nil)), "/login")); !fellBack(landed) {
		t.Fatalf("a later sign-in went to %s, want the second-factor page", landed)
	}
	again.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if offer.SecondFactor.Enrol || !strings.Contains(strings.Join(offer.SecondFactor.Methods, " "), "totp") {
		t.Errorf("a later sign-in's second-factor page = %+v, want the authenticator asked for", offer.SecondFactor)
	}
}

// In provider mode an OpenID Connect provider's amr that holds an accepted
// value marks the session as having passed a second factor: a requiring
// application takes it, without an amr of this server's own, and its refresh
// family goes on. One that does not falls back to the account's own factor,
// and the fallback is logged.
func TestLiveAProviderMayAnswerForTheSecondFactor(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)
	idp := newFakeOIDC(t)
	id := oidcConnection(t, f.super, idp, map[string]any{"create_users": true, "second_factor": "provider"})

	tests := []struct {
		name     string
		accepted []string
		amr      any
		proved   bool
	}{
		{"amr holding mfa", nil, []string{"pwd", "mfa"}, true},
		{"amr without an accepted value", nil, []string{"pwd"}, false},
		{"no amr at all", nil, nil, false},
		{"amr as a bare string, not a list", nil, "mfa", false},
		{"mfa, where only hwk is accepted", []string{"hwk"}, []string{"pwd", "mfa"}, false},
		{"hwk, where hwk is accepted", []string{"hwk"}, []string{"hwk"}, true},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accepted := tt.accepted
			if accepted == nil {
				accepted = []string{"mfa"}
			}
			f.super.must(http.StatusOK, http.MethodPatch, "/sso-connections/"+id, map[string]any{"accepted_amr": accepted}, nil)

			email := "person" + string(rune('a'+i)) + "@acme.test"
			claims := map[string]any{"sub": "okta|" + email, "email": email, "email_verified": true}
			if tt.amr != nil {
				claims["amr"] = tt.amr
			}
			idp.signs(claims)

			b := f.s.browser()
			verifier, challenge := pkce(t)
			landed := idp.ssoInto(b, "acme", page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"))

			if !tt.proved {
				if !fellBack(landed) {
					t.Fatalf("went to %s, want the account's own factor asked for", landed)
				}
				var logs struct {
					Logs []struct {
						Action string `json:"action"`
						Actor  string `json:"actor"`
					} `json:"logs"`
				}
				f.super.must(http.StatusOK, http.MethodGet, "/logs?action=user.sso_second_factor_fallback", nil, &logs)
				found := false
				for _, entry := range logs.Logs {
					found = found || entry.Actor == email
				}
				if !found {
					t.Errorf("the fallback for %s was not logged: %+v", email, logs.Logs)
				}
				return
			}

			tokens := vault.exchange(codeBack(t, landed, vaultRedirect), verifier)
			if tokens.status != http.StatusOK {
				t.Fatalf("exchanging the code = %d %v", tokens.status, tokens.body)
			}
			if amr := amrOf(t, tokens); len(amr) != 0 {
				t.Errorf("amr = %v, want none: what the provider proved is its own to say", amr)
			}
			refreshed := vault.f.token("/oauth2/token", vault.clientID, vault.secret, url.Values{
				"grant_type": {"refresh_token"}, "refresh_token": {tokens.str("refresh_token")},
			})
			if refreshed.status != http.StatusOK {
				t.Errorf("refreshing under the requiring flow = %d %v", refreshed.status, refreshed.body)
			}

			// The session passed a second factor: the vault asks nothing more.
			_, challenge = pkce(t)
			codeBack(t, b.visit(vault.authorizeURL(challenge, nil)), vaultRedirect)
		})
	}
}

// Userinfo is unsigned, and fills in only what the id_token left out: an amr
// it offers is not the provider's word, so it never counts.
func TestLiveAnAMRFromUserinfoIsNotBelieved(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)
	idp := newFakeOIDC(t)
	idp.userinfo = map[string]any{"sub": "okta|lin", "email": "lin@acme.test", "email_verified": true, "amr": []string{"mfa"}}
	oidcConnection(t, f.super, idp, map[string]any{"create_users": true, "second_factor": "provider"})
	idp.signs(map[string]any{"sub": "okta|lin"})

	b := f.s.browser()
	_, challenge := pkce(t)
	landed := idp.ssoInto(b, "acme", page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"))
	if !fellBack(landed) {
		t.Fatalf("an amr from userinfo went to %s, want the account's own factor asked for", landed)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("the sign-in opened a session on userinfo's amr")
	}
}

// A SAML connection in provider mode counts the authentication contexts it
// lists, read from the assertion the provider signed and nowhere else: a
// context elsewhere in the response, even one the provider signed, is not
// believed.
func TestLiveASAMLProviderAnswersOnlyInItsSignedAssertion(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)

	idp := newFakeSAML(t)
	var created struct {
		Connection idOnly `json:"connection"`
	}
	f.super.must(http.StatusCreated, http.MethodPost, "/sso-connections", map[string]any{
		"protocol": "saml", "name": "Globex", "slug": "globex", "metadata": idp.metadata(t),
		"domains": []string{"globex.test"}, "is_enabled": true, "create_users": true,
		"second_factor": "provider", "mfa_contexts": []string{samlMFA},
	}, &created)

	signs := func(email, class string) {
		idp.mu.Lock()
		defer idp.mu.Unlock()
		idp.idp.AssertionMaker = classMaker{class: class}
		idp.person = &saml.Session{
			ID: "session-" + email, CreateTime: time.Now(), ExpireTime: time.Now().Add(time.Hour),
			NameID: email, UserEmail: email, UserGivenName: "Hedy",
		}
	}
	strayClass := func(response *etree.Element) {
		statement := response.CreateElement("saml:AuthnStatement")
		statement.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
		statement.CreateAttr("AuthnInstant", time.Now().UTC().Format(time.RFC3339))
		statement.CreateElement("saml:AuthnContext").CreateElement("saml:AuthnContextClassRef").SetText(samlMFA)
	}

	tests := []struct {
		name   string
		class  string
		change func(*etree.Element)
		proved bool
	}{
		{"a listed context in the signed assertion", samlMFA, nil, true},
		{"a context it does not list", samlPassword, nil, false},
		// The control: a response signed again as a whole is still taken.
		{"a listed context, the response signed again", samlMFA, func(*etree.Element) {}, true},
		{"a listed context outside the assertion, the response signed again", samlPassword, strayClass, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := "hedy" + string(rune('a'+i)) + "@globex.test"
			signs(email, tt.class)

			b := f.s.browser()
			verifier, challenge := pkce(t)
			form := idp.samlForm(t, b, "globex", page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"))
			if tt.change != nil {
				form = idp.resigned(t, form, tt.change)
			}
			result := b.post(t, f.s.root+"/oauth2/sso/globex/acs", form)
			if reason, failed := result.failed(); failed {
				var logs struct {
					Logs []struct {
						Detail string `json:"detail"`
					} `json:"logs"`
				}
				f.super.must(http.StatusOK, http.MethodGet, "/logs?action=sso_connection.sign_in_failed&limit=1", nil, &logs)
				t.Fatalf("the sign-in failed: %s %+v", reason, logs.Logs)
			}

			if !tt.proved {
				if !fellBack(result.landed) {
					t.Errorf("went to %s, want the account's own factor asked for", result.landed)
				}
				return
			}
			tokens := vault.exchange(codeBack(t, result.landed, vaultRedirect), verifier)
			if tokens.status != http.StatusOK {
				t.Errorf("exchanging the code = %d %v", tokens.status, tokens.body)
			}
		})
	}
}

// The first link of a connection to an existing account with a factor waits
// for that factor under a requiring flow too, even when the provider's answer
// shows one of its own: the link is held on the waiting sign-in, and made
// only once the account's factor is answered.
func TestLiveAFirstLinkStillWaitsForTheAccountsFactor(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")
	vault := f.requiringApp(t)
	idp := newFakeOIDC(t)

	const email = "ada@acme.test"
	f.makeUser(email, adaPassword, true)
	owner := f.s.browser()
	if status := owner.signInAs(email, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("Ada's own sign-in = %d", status)
	}
	secret, _ := addAuthenticator(t, owner)

	id := oidcConnection(t, f.super, idp, map[string]any{"second_factor": "provider"})
	f.s.provedDomain(t, id, "acme.test")
	idp.signs(map[string]any{"sub": "okta|ada", "email": email, "email_verified": true, "amr": []string{"pwd", "mfa"}})

	b := f.s.browser()
	verifier, challenge := pkce(t)
	landed := idp.ssoInto(b, "acme", page(t, b.visit(vault.authorizeURL(challenge, nil)), "/login"))
	if !fellBack(landed) {
		t.Fatalf("the first link went to %s, want the account's own factor asked for", landed)
	}
	var linking int64
	if err := f.s.db.Raw("SELECT count(*) FROM pending_sign_ins WHERE link_connection_id = ? AND used_at IS NULL", id).Scan(&linking).Error; err != nil {
		t.Fatal(err)
	}
	if linking != 1 {
		t.Errorf("%d waiting sign-ins hold the link, want 1", linking)
	}
	if links := f.s.count(t, "sso_identities"); links != 0 {
		t.Fatalf("%d identities were linked before the account's factor", links)
	}

	status, to := answerSecondFactor(b, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))})
	if status != http.StatusOK {
		t.Fatalf("Ada's code = %d %s", status, to)
	}
	if links := f.s.count(t, "sso_identities"); links != 1 {
		t.Errorf("%d identities linked after the account's factor, want 1", links)
	}
	back, _ := url.Parse(to)
	vault.exchange(codeBack(t, back, vaultRedirect), verifier)
	if b.cookieOf(session.SecondFactorCookie, "/") != "" {
		t.Error("the waiting sign-in's cookie was kept")
	}
}
