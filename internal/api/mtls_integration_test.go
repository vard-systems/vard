package api

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vard/internal/api/kit/clientcert"
	"vard/internal/brand"
	"vard/internal/config"
	"vard/internal/model"
	"vard/internal/model/certtest"
)

// These tests run the server with mutual TLS on (RFC 8705), the way it runs
// behind nginx: the proxy asks for a client certificate on the mutual TLS
// host and the id app passes it on in clientcert.Header, which the API
// believes from the apps alone — here, from this machine.

const mtlsAlias = "https://mtls.test"

type mtlsFixture struct {
	*oauthFixture
	authority *certtest.Authority
}

func newMTLSFixture(t *testing.T) *mtlsFixture {
	authority := certtest.NewAuthority(t, "Clients CA")
	file := filepath.Join(t.TempDir(), "clients-ca.pem")
	if err := os.WriteFile(file, []byte(certtest.PEM(authority.Certificate)), 0o600); err != nil {
		t.Fatal(err)
	}

	f := newOAuthFixtureWith(t, func(cfg *config.Config) {
		cfg.MTLSURL = mtlsAlias
		cfg.MTLSCAFile = file
		cfg.TrustedProxies = []string{"127.0.0.1/32", "::1/128"}
	})

	return &mtlsFixture{oauthFixture: f, authority: authority}
}

// send posts a form to the provider the way the mutual TLS host passes it
// on: the client id in the form, the certificate in the header, and a
// secret by HTTP Basic when one is given.
func (f *mtlsFixture) send(path, clientID, secret string, cert *x509.Certificate, form url.Values) tokenResult {
	f.s.t.Helper()

	if secret == "" {
		form.Set("client_id", clientID)
	}
	req, err := http.NewRequest(http.MethodPost, f.s.root+path, strings.NewReader(form.Encode()))
	if err != nil {
		f.s.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if secret != "" {
		req.SetBasicAuth(clientID, secret)
	}
	if cert != nil {
		req.Header.Set(clientcert.Header, certtest.Header(cert))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		f.s.t.Fatal(err)
	}
	defer res.Body.Close()

	result := tokenResult{status: res.StatusCode, body: map[string]any{}}
	_ = json.NewDecoder(res.Body).Decode(&result.body)
	return result
}

// with calls a URL with an access token, over a connection with the
// certificate when one is given.
func (f *mtlsFixture) with(target, token string, cert *x509.Certificate) (int, map[string]any) {
	f.s.t.Helper()

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		f.s.t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if cert != nil {
		req.Header.Set(clientcert.Header, certtest.Header(cert))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		f.s.t.Fatal(err)
	}
	defer res.Body.Close()

	out := map[string]any{}
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// as is the fixture's sign-in for another application with the same
// redirect URI.
func (f *mtlsFixture) as(clientID string) *oauthFixture {
	other := *f.oauthFixture
	other.clientID, other.clientSecret = clientID, ""
	return &other
}

func confirmedCertificate(t *testing.T, f *oauthFixture, token string) string {
	t.Helper()
	_, claims := f.verifyJWT(token)
	cnf, _ := claims["cnf"].(map[string]any)
	thumbprint, _ := cnf["x5t#S256"].(string)
	return thumbprint
}

// The discovery document says where mutual TLS is and what it offers, and
// says nothing of it on a server without it.
func TestLiveMutualTLSDiscovery(t *testing.T) {
	read := func(f *oauthFixture) map[string]any {
		t.Helper()
		res, err := http.Get(f.s.root + "/.well-known/openid-configuration")
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		out := map[string]any{}
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out
	}

	on := read(newMTLSFixture(t).oauthFixture)
	aliases, _ := on["mtls_endpoint_aliases"].(map[string]any)
	for name, path := range map[string]string{
		"token_endpoint": "/oauth2/token", "userinfo_endpoint": "/oauth2/userinfo",
		"pushed_authorization_request_endpoint": "/oauth2/par", "revocation_endpoint": "/oauth2/revoke",
		"introspection_endpoint": "/oauth2/introspect", "device_authorization_endpoint": "/oauth2/device_authorization",
	} {
		if aliases[name] != mtlsAlias+path {
			t.Errorf("mtls_endpoint_aliases.%s = %v, want %s", name, aliases[name], mtlsAlias+path)
		}
	}
	if on["tls_client_certificate_bound_access_tokens"] != true {
		t.Error("tls_client_certificate_bound_access_tokens is not true with mutual TLS on")
	}
	methods := stringList(on["token_endpoint_auth_methods_supported"])
	for _, method := range []string{"tls_client_auth", "self_signed_tls_client_auth"} {
		if !strings.Contains(strings.Join(methods, " "), method) {
			t.Errorf("token_endpoint_auth_methods_supported = %v, want %s", methods, method)
		}
	}

	off := read(newOAuthFixture(t))
	if _, ok := off["mtls_endpoint_aliases"]; ok || off["tls_client_certificate_bound_access_tokens"] != false {
		t.Errorf("without mutual TLS, discovery says aliases %v and bound tokens %v", off["mtls_endpoint_aliases"], off["tls_client_certificate_bound_access_tokens"])
	}
	if strings.Contains(strings.Join(stringList(off["token_endpoint_auth_methods_supported"]), " "), "tls") {
		t.Errorf("without mutual TLS, token_endpoint_auth_methods_supported = %v", off["token_endpoint_auth_methods_supported"])
	}
}

// A service that registered a self-signed certificate proves itself with
// exactly that certificate and nothing else, and its tokens are bound to it.
func TestLiveMutualTLSSelfSignedService(t *testing.T) {
	f := newMTLSFixture(t)
	registered := certtest.SelfSigned(t, "nightly-job")

	clientID, secret, appID := f.register(map[string]any{
		"name": "Nightly job", "type": "m2m",
		"token_auth_method":                          "self_signed_tls_client_auth",
		"tls_certificates":                           []string{certtest.PEM(registered)},
		"tls_client_certificate_bound_access_tokens": true,
	})
	if secret != "" {
		t.Errorf("a self-signed client was given a secret")
	}
	f.authorizeAPI(appID, "orders:read")

	ask := func() url.Values {
		return url.Values{"grant_type": {"client_credentials"}, "audience": {ordersAPI}}
	}

	tests := []struct {
		name string
		cert *x509.Certificate
		want int
	}{
		{"with no certificate", nil, http.StatusUnauthorized},
		{"with another self-signed certificate of the same name", certtest.SelfSigned(t, "nightly-job"), http.StatusUnauthorized},
		{"with one from a trusted authority", f.authority.Issue(t, certtest.Subject{CommonName: "nightly-job"}), http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.send("/oauth2/token", clientID, "", tt.cert, ask()); got.status != tt.want || got.str("error") != "invalid_client" {
				t.Errorf("token = %d %v, want %d invalid_client", got.status, got.body, tt.want)
			}
		})
	}

	if got := f.send("/oauth2/token", clientID, "a-secret", registered, ask()); got.str("error") != "invalid_client" {
		t.Errorf("token with a secret beside the certificate = %d %v, want invalid_client", got.status, got.body)
	}

	tokens := f.send("/oauth2/token", clientID, "", registered, ask())
	if tokens.status != http.StatusOK {
		t.Fatalf("token with the registered certificate = %d %v", tokens.status, tokens.body)
	}
	if got := confirmedCertificate(t, f.oauthFixture, tokens.str("access_token")); got != model.CertificateThumbprint(registered) {
		t.Errorf("cnf.x5t#S256 = %q, want the registered certificate's %q", got, model.CertificateThumbprint(registered))
	}
}

// A person's tokens for a tls_client_auth application are bound to its
// certificate: userinfo takes them only over a connection with it, and the
// account API, which checks no certificate, not at all.
func TestLiveMutualTLSBindsAPersonsTokens(t *testing.T) {
	f := newMTLSFixture(t)

	clientID, _, appID := f.register(map[string]any{
		"name": "Partner portal", "type": "web",
		"grant_types":       []string{"authorization_code", "refresh_token"},
		"redirect_uris":     []string{shopRedirect},
		"scopes":            []string{"openid", "email", "offline_access"},
		"token_auth_method": "tls_client_auth",
		"tls_subject_kind":  "san_dns",
		"tls_subject":       "portal.example.com",
		"tls_client_certificate_bound_access_tokens": true,
	})
	accountAPI, scopes := f.systemAPI(model.SystemAccountAPI)
	f.allow(appID, accountAPI, scopes, model.ScopeAccountRead)

	portal := f.authority.Issue(t, certtest.Subject{CommonName: "portal", DNS: []string{"portal.example.com"}})
	impostor := f.authority.Issue(t, certtest.Subject{CommonName: "portal", DNS: []string{"impostor.example.com"}})
	untrusted := certtest.NewAuthority(t, "Not ours").Issue(t, certtest.Subject{CommonName: "portal", DNS: []string{"portal.example.com"}})

	verifier, challenge := pkce(t)
	back := f.as(clientID).signInWith(f.s.browser(), challenge, url.Values{
		"scope": {"openid email offline_access " + model.ScopeAccountRead}, "audience": {brand.AccountAPIIdentifier},
	})
	exchange := func(cert *x509.Certificate) tokenResult {
		return f.send("/oauth2/token", clientID, "", cert, url.Values{
			"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
			"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
		})
	}

	for name, cert := range map[string]*x509.Certificate{"no certificate": nil, "another subject": impostor, "an authority not trusted": untrusted} {
		if got := exchange(cert); got.status != http.StatusUnauthorized || got.str("error") != "invalid_client" {
			t.Errorf("exchange with %s = %d %v, want invalid_client", name, got.status, got.body)
		}
	}

	tokens := exchange(portal)
	if tokens.status != http.StatusOK {
		t.Fatalf("exchange with the portal's certificate = %d %v", tokens.status, tokens.body)
	}
	access := tokens.str("access_token")
	if got := confirmedCertificate(t, f.oauthFixture, access); got != model.CertificateThumbprint(portal) {
		t.Errorf("cnf.x5t#S256 = %q, want the portal's certificate", got)
	}
	if tokens.str("token_type") != "Bearer" {
		t.Errorf("token_type = %q, want Bearer: a certificate-bound token is still sent as one", tokens.str("token_type"))
	}

	userinfo := f.s.root + "/oauth2/userinfo"
	if status, body := f.with(userinfo, access, nil); status != http.StatusUnauthorized || body["error"] != "invalid_token" {
		t.Errorf("userinfo with the token alone = %d %v, want invalid_token", status, body)
	}
	if status, _ := f.with(userinfo, access, impostor); status != http.StatusUnauthorized {
		t.Errorf("userinfo over another certificate = %d, want 401", status)
	}
	if status, body := f.with(userinfo, access, portal); status != http.StatusOK || body["email"] != adaEmail {
		t.Errorf("userinfo over the portal's certificate = %d %v", status, body)
	}

	if status, body := f.with(f.s.root+"/api/v1/account/me", access, portal); status != http.StatusUnauthorized || body["code"] != "account_token_refused" {
		t.Errorf("the account API with a certificate-bound token = %d %v, want account_token_refused", status, body)
	}
}

// A public client has no credentials, but may bind its tokens to a
// certificate it holds; its refresh tokens are then bound too (RFC 8705 §4),
// so a copied one is no use over another connection.
func TestLiveMutualTLSBindsAPublicClientsRefreshTokens(t *testing.T) {
	f := newMTLSFixture(t)

	clientID, _, _ := f.register(map[string]any{
		"name": "Kiosk", "type": "native",
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{shopRedirect},
		"scopes":        []string{"openid", "offline_access"},
		"tls_client_certificate_bound_access_tokens": true,
	})
	held := certtest.SelfSigned(t, "kiosk")
	other := certtest.SelfSigned(t, "kiosk")

	verifier, challenge := pkce(t)
	back := f.as(clientID).signInWith(f.s.browser(), challenge, url.Values{"scope": {"openid offline_access"}, "audience": {""}})
	exchange := url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
		"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
	}

	if got := f.send("/oauth2/token", clientID, "", nil, exchange); got.status != http.StatusBadRequest || got.str("error") != "invalid_request" {
		t.Errorf("exchange without a certificate = %d %v, want invalid_request", got.status, got.body)
	}
	tokens := f.send("/oauth2/token", clientID, "", held, exchange)
	if tokens.status != http.StatusOK || tokens.str("refresh_token") == "" {
		t.Fatalf("exchange with a certificate = %d %v", tokens.status, tokens.body)
	}

	refresh := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {tokens.str("refresh_token")}}
	if got := f.send("/oauth2/token", clientID, "", other, refresh); got.status != http.StatusBadRequest || got.str("error") != "invalid_grant" {
		t.Errorf("refresh over another certificate = %d %v, want invalid_grant", got.status, got.body)
	}
	if got := f.send("/oauth2/token", clientID, "", held, refresh); got.status != http.StatusOK {
		t.Errorf("refresh over the certificate it is bound to = %d %v", got.status, got.body)
	}
}

// A DPoP proof may name the token endpoint at its mutual TLS alias, which is
// where the client sent it; any other address is still refused.
func TestLiveMutualTLSDPoPAtTheAlias(t *testing.T) {
	f := newMTLSFixture(t)
	spaID, _, _ := f.register(map[string]any{
		"name": "Storefront", "type": "spa",
		"grant_types":   []string{"authorization_code"},
		"redirect_uris": []string{shopRedirect},
		"scopes":        []string{"openid"},
	})
	key := newDPoPKey(t)

	signIn := func() url.Values {
		verifier, challenge := pkce(t)
		back := f.as(spaID).signInWith(f.s.browser(), challenge, url.Values{"scope": {"openid"}, "audience": {""}})
		return url.Values{
			"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
			"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
		}
	}

	if got := postToken(t, f.oauthFixture, spaID, key.proof(t, http.MethodPost, mtlsAlias+"/oauth2/token", ""), signIn()); got.status != http.StatusOK || got.str("token_type") != "DPoP" {
		t.Errorf("a proof naming the alias = %d %v, want DPoP tokens", got.status, got.body)
	}
	if got := postToken(t, f.oauthFixture, spaID, key.proof(t, http.MethodPost, "https://elsewhere.test/oauth2/token", ""), signIn()); got.str("error") != "invalid_dpop_proof" {
		t.Errorf("a proof naming another host = %d %v, want invalid_dpop_proof", got.status, got.body)
	}
}

// What needs mutual TLS is refused where the server does not offer it: the
// application could never get a token.
func TestLiveMutualTLSIsRefusedWhereItIsOff(t *testing.T) {
	cert := certtest.PEM(certtest.SelfSigned(t, "job"))

	off := newOAuthFixture(t)
	for name, body := range map[string]map[string]any{
		"a self-signed client": {"name": "Job", "type": "m2m", "token_auth_method": "self_signed_tls_client_auth", "tls_certificates": []string{cert}},
		"bound tokens":         {"name": "Job", "type": "m2m", "tls_client_certificate_bound_access_tokens": true},
	} {
		var out map[string]any
		if status := off.super.do(http.MethodPost, "/applications", body, &out); status != http.StatusBadRequest || out["code"] != "application_mtls_off" {
			t.Errorf("%s without mutual TLS = %d %v, want application_mtls_off", name, status, out)
		}
	}

	// With mutual TLS but no authorities, only self-signed certificates.
	selfSignedOnly := newOAuthFixtureWith(t, func(cfg *config.Config) { cfg.MTLSURL = mtlsAlias })
	var out map[string]any
	status := selfSignedOnly.super.do(http.MethodPost, "/applications", map[string]any{
		"name": "Job", "type": "m2m", "token_auth_method": "tls_client_auth", "tls_subject_kind": "subject_dn", "tls_subject": "CN=job",
	}, &out)
	if status != http.StatusBadRequest || out["code"] != "application_tls_authorities_missing" {
		t.Errorf("tls_client_auth without authorities = %d %v, want application_tls_authorities_missing", status, out)
	}
}

// A client registers its certificate the way RFC 8705 and RFC 7591 say: in
// its jwks as x5c, or the one subject its CA-issued certificate carries.
func TestLiveMutualTLSRegistration(t *testing.T) {
	f := newMTLSFixture(t)

	cliID := f.adminCLI()
	adminToken := f.adminCLITokens(cliID)
	apiID, scopes := f.systemAPI(model.SystemAdminAPI)
	f.allow(cliID, apiID, scopes, model.PermApplicationsWrite)
	token := adminToken()

	cert := certtest.SelfSigned(t, "partner")
	status, body := f.bearer(http.MethodPost, f.s.root+"/oauth2/register", token, map[string]any{
		"client_name": "Partner job", "grant_types": []string{"client_credentials"},
		"token_endpoint_auth_method": "self_signed_tls_client_auth",
		"jwks":                       map[string]any{"keys": []map[string]any{{"x5c": []string{base64.StdEncoding.EncodeToString(cert.Raw)}}}},
		"tls_client_certificate_bound_access_tokens": true,
	})
	if status != http.StatusCreated || body["client_secret"] != nil || body["tls_client_certificate_bound_access_tokens"] != true || body["jwks"] == nil {
		t.Errorf("registering a self-signed client = %d %v", status, body)
	}

	status, body = f.bearer(http.MethodPost, f.s.root+"/oauth2/register", token, map[string]any{
		"client_name": "Partner portal", "redirect_uris": []string{"https://partner.example.com/callback"},
		"token_endpoint_auth_method": "tls_client_auth", "tls_client_auth_san_dns": "partner.example.com",
	})
	if status != http.StatusCreated || body["tls_client_auth_san_dns"] != "partner.example.com" {
		t.Errorf("registering a tls_client_auth client = %d %v", status, body)
	}

	status, body = f.bearer(http.MethodPost, f.s.root+"/oauth2/register", token, map[string]any{
		"client_name": "Two names", "redirect_uris": []string{"https://partner.example.com/callback"},
		"token_endpoint_auth_method": "tls_client_auth",
		"tls_client_auth_san_dns":    "partner.example.com", "tls_client_auth_subject_dn": "CN=partner",
	})
	if status != http.StatusBadRequest || body["error"] != "invalid_client_metadata" {
		t.Errorf("registering with two subjects = %d %v, want invalid_client_metadata", status, body)
	}
}

// A certificate is public, so the header naming one is worth only as much as
// whoever set it. With mutual TLS off, or from anyone but the apps, the API
// ignores it: a caller holding a copy of a client's certificate, but not its
// key, gets neither the client's tokens nor the use of its bound ones.
func TestLiveMutualTLSIgnoresAForgedCertificateHeader(t *testing.T) {
	cert := certtest.SelfSigned(t, "nightly-job")

	selfSigned := func(f *oauthFixture) string {
		t.Helper()
		// Made in the store, past the panel, which would refuse it without
		// mutual TLS: as a server that turned mutual TLS off would have it.
		app := &model.Application{
			Name: "Nightly job", Type: model.AppM2M, ClientID: "nightly-" + randomHex(t, 4),
			TokenAuthMethod: model.AuthSelfSignedTLSClient, TLSCertificates: []string{certtest.PEM(cert)},
			GrantTypes:          []string{model.GrantClientCredentials},
			AccessTokenLifetime: model.DefaultAccessTokenLifetime, IDTokenLifetime: model.DefaultIDTokenLifetime,
			RefreshTokenLifetime: model.DefaultRefreshTokenLifetime, IsEnabled: true,
		}
		if err := f.s.store.CreateApplication(t.Context(), app); err != nil {
			t.Fatal(err)
		}
		f.authorizeAPI(app.ID.String(), "orders:read")
		return app.ClientID
	}

	servers := map[string]func(*config.Config){
		"with mutual TLS off": func(cfg *config.Config) {
			cfg.TrustedProxies = []string{"127.0.0.1/32", "::1/128"}
		},
		"from a caller that is not one of the apps": func(cfg *config.Config) {
			cfg.MTLSURL = mtlsAlias
			cfg.TrustedProxies = []string{"192.0.2.1/32"}
		},
	}

	for name, change := range servers {
		t.Run(name, func(t *testing.T) {
			f := &mtlsFixture{oauthFixture: newOAuthFixtureWith(t, change)}
			clientID := selfSigned(f.oauthFixture)

			got := f.send("/oauth2/token", clientID, "", cert, url.Values{"grant_type": {"client_credentials"}, "audience": {ordersAPI}})
			if got.status != http.StatusUnauthorized || got.str("error") != "invalid_client" {
				t.Errorf("token with a forged certificate header = %d %v, want invalid_client", got.status, got.body)
			}
		})
	}
}

// A retry within the grace from a public client whose tokens are bound to
// its certificate must come over a connection with that certificate, like
// any refresh; from another, or none, it is refused, not handed the sealed
// token.
func TestLiveMutualTLSRefreshGraceKeepsTheCertificate(t *testing.T) {
	f := newMTLSFixture(t)
	clientID, _, _ := f.register(map[string]any{
		"name": "Kiosk", "type": "native",
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{shopRedirect},
		"scopes":        []string{"openid", "offline_access"},
		"tls_client_certificate_bound_access_tokens": true,
	})
	held := certtest.SelfSigned(t, "kiosk")

	verifier, challenge := pkce(t)
	back := f.as(clientID).signInWith(f.s.browser(), challenge, url.Values{"scope": {"openid offline_access"}, "audience": {""}})
	r1 := f.send("/oauth2/token", clientID, "", held, url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
		"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
	}).str("refresh_token")
	retry := func(cert *x509.Certificate) tokenResult {
		return f.send("/oauth2/token", clientID, "", cert, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {r1}})
	}

	r2 := retry(held).str("refresh_token")
	if r2 == "" {
		t.Fatal("the first refresh gave no token")
	}

	if got := retry(certtest.SelfSigned(t, "kiosk")); got.str("refresh_token") != "" || got.str("error") != "invalid_grant" {
		t.Errorf("a retry over another certificate = %d %v, want invalid_grant", got.status, got.body)
	}
	if got := retry(nil); got.str("refresh_token") != "" || got.str("error") != "invalid_request" {
		t.Errorf("a retry with no certificate = %d %v, want invalid_request", got.status, got.body)
	}
	if got := retry(held); got.str("refresh_token") != r2 {
		t.Errorf("the retry over the bound certificate = %d %v, want the token it already got", got.status, got.body)
	}
}
