package model_test

import (
	"crypto/x509"
	"strings"
	"testing"
	"time"

	"vard/internal/model"
	"vard/internal/model/certtest"
)

// A tls_client_auth client is recognised by a certificate from an authority
// the server trusts, carrying exactly the subject it registered; a
// self-signed one by the very certificate it registered. Anything else is
// somebody else.
func TestApplicationMatchesCertificate(t *testing.T) {
	trusted := certtest.NewAuthority(t, "Trusted CA")
	stranger := certtest.NewAuthority(t, "Some other CA")

	issued := trusted.Issue(t, certtest.Subject{
		CommonName: "shop",
		DNS:        []string{"shop.example.com"},
		URIs:       []string{"spiffe://example.com/shop"},
		IPs:        []string{"192.0.2.10"},
		Emails:     []string{"shop@example.com"},
	})
	expired := trusted.Issue(t, certtest.Subject{CommonName: "shop", NotAfter: time.Now().Add(-time.Minute)})
	elsewhere := stranger.Issue(t, certtest.Subject{CommonName: "shop", DNS: []string{"shop.example.com"}})

	registered := certtest.SelfSigned(t, "shop")
	another := certtest.SelfSigned(t, "shop")

	pki := func(kind model.TLSSubjectKind, subject string) model.Application {
		return model.Application{TokenAuthMethod: model.AuthTLSClient, TLSSubjectKind: kind, TLSSubject: subject}
	}
	selfSigned := model.Application{TokenAuthMethod: model.AuthSelfSignedTLSClient, TLSCertificates: []string{certtest.PEM(registered)}}

	tests := []struct {
		name string
		app  model.Application
		cert *x509.Certificate
		want bool
	}{
		{"the subject DN", pki(model.TLSSubjectDN, "CN=shop,O=Example"), issued, true},
		{"the DNS name", pki(model.TLSSubjectDNS, "shop.example.com"), issued, true},
		{"the DNS name in capitals", pki(model.TLSSubjectDNS, "SHOP.example.com"), issued, true},
		{"the URI", pki(model.TLSSubjectURI, "spiffe://example.com/shop"), issued, true},
		{"the IP address", pki(model.TLSSubjectIP, "192.0.2.10"), issued, true},
		{"the email address", pki(model.TLSSubjectEmail, "shop@example.com"), issued, true},
		{"another DN", pki(model.TLSSubjectDN, "CN=shop,O=Other"), issued, false},
		{"another DNS name", pki(model.TLSSubjectDNS, "admin.example.com"), issued, false},
		{"a DNS name only ending the same", pki(model.TLSSubjectDNS, "example.com"), issued, false},
		{"the right name from an authority not trusted", pki(model.TLSSubjectDNS, "shop.example.com"), elsewhere, false},
		{"an expired certificate", pki(model.TLSSubjectDN, "CN=shop,O=Example"), expired, false},
		{"a self-signed certificate for tls_client_auth", pki(model.TLSSubjectDN, "CN=shop,O=Example"), registered, false},
		{"the registered self-signed certificate", selfSigned, registered, true},
		{"another self-signed certificate with the same name", selfSigned, another, false},
		{"a CA-issued certificate for a self-signed client", selfSigned, issued, false},
		{"no certificate at all", pki(model.TLSSubjectDN, "CN=shop,O=Example"), nil, false},
		{"a secret client", model.Application{TokenAuthMethod: model.AuthClientSecretBasic}, issued, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.app.MatchesCertificate(tt.cert, trusted.Pool(), time.Now()); got != tt.want {
				t.Errorf("MatchesCertificate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Mutual TLS settings are checked as the panel and registration send them.
func TestApplicationValidatesCertificateSettings(t *testing.T) {
	cert := certtest.PEM(certtest.SelfSigned(t, "shop"))

	tests := []struct {
		name   string
		kind   model.ApplicationType
		change func(*model.Application)
		want   string
	}{
		{"tls_client_auth with a DN", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSSubjectKind, a.TLSSubject = model.AuthTLSClient, model.TLSSubjectDN, "CN=shop"
		}, ""},
		{"tls_client_auth for a service", model.AppM2M, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSSubjectKind, a.TLSSubject = model.AuthTLSClient, model.TLSSubjectDNS, "svc.example.com"
		}, ""},
		{"tls_client_auth with no subject", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSSubjectKind = model.AuthTLSClient, model.TLSSubjectDN
		}, "tls_subject"},
		{"tls_client_auth with an unknown kind of subject", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSSubjectKind, a.TLSSubject = model.AuthTLSClient, "serial", "1"
		}, "tls_subject_kind"},
		{"tls_client_auth with an IP that is none", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSSubjectKind, a.TLSSubject = model.AuthTLSClient, model.TLSSubjectIP, "shop"
		}, "not an IP address"},
		{"a self-signed client with its certificate", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSCertificates = model.AuthSelfSignedTLSClient, []string{cert}
		}, ""},
		{"a self-signed client without one", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod = model.AuthSelfSignedTLSClient
		}, "needs the client's certificate"},
		{"a self-signed client with something that is no certificate", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod, a.TLSCertificates = model.AuthSelfSignedTLSClient, []string{"-----BEGIN CERTIFICATE-----\nnope\n-----END CERTIFICATE-----"}
		}, "not a PEM certificate"},
		{"a self-signed client with too many", model.AppWeb, func(a *model.Application) {
			a.TokenAuthMethod = model.AuthSelfSignedTLSClient
			for range 5 {
				a.TLSCertificates = append(a.TLSCertificates, certtest.PEM(certtest.SelfSigned(t, "shop")))
			}
		}, "at most"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := model.Application{
				Name:                 "Shop",
				Type:                 tt.kind,
				GrantTypes:           []string{model.GrantAuthorizationCode},
				RedirectURIs:         []string{"https://shop.example.com/callback"},
				AccessTokenLifetime:  model.DefaultAccessTokenLifetime,
				IDTokenLifetime:      model.DefaultIDTokenLifetime,
				RefreshTokenLifetime: model.DefaultRefreshTokenLifetime,
			}
			tt.change(&a)
			a.Normalize()

			err := a.Validate()
			if tt.want == "" && err != nil {
				t.Fatalf("Validate() = %v, want nothing", err)
			}
			if tt.want != "" && (err == nil || !strings.Contains(err.Error(), tt.want)) {
				t.Fatalf("Validate() = %v, want it to say %q", err, tt.want)
			}
			if a.TokenAuthMethod.UsesCertificate() && a.HasSecret() {
				t.Error("HasSecret() = true for a certificate method; it would be given a secret")
			}
		})
	}
}

// A public client is normalized onto none, so a certificate method it sent
// is dropped along with the certificate settings.
func TestPublicClientsDropCertificateMethods(t *testing.T) {
	a := model.Application{Type: model.AppSPA, TokenAuthMethod: model.AuthSelfSignedTLSClient, TLSCertificates: []string{"x"}}
	a.Normalize()

	if a.TokenAuthMethod != model.AuthNone || len(a.TLSCertificates) != 0 || !a.MustUsePKCE() {
		t.Errorf("method %s, certificates %v, PKCE %v; want none, none and PKCE", a.TokenAuthMethod, a.TLSCertificates, a.MustUsePKCE())
	}
}
