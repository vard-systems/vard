// Package certtest makes throwaway certificates for tests of mutual TLS: an
// authority, client certificates it issues, and self-signed ones.
package certtest

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"
)

// Authority is a certificate authority that issues client certificates.
type Authority struct {
	Certificate *x509.Certificate
	key         *ecdsa.PrivateKey
}

// Subject is what a client certificate names.
type Subject struct {
	CommonName string
	DNS        []string
	URIs       []string
	IPs        []string
	Emails     []string
	// NotAfter is when it expires; zero is in a day.
	NotAfter time.Time
}

func NewAuthority(t testing.TB, name string) *Authority {
	t.Helper()

	key := newKey(t)
	template := &x509.Certificate{
		SerialNumber:          serial(t),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	return &Authority{Certificate: sign(t, template, template, &key.PublicKey, key), key: key}
}

// Pool is the authority alone, as VARD_MTLS_CA_FILE would hold it.
func (a *Authority) Pool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(a.Certificate)
	return pool
}

// Issue makes a client certificate for the subject.
func (a *Authority) Issue(t testing.TB, subject Subject) *x509.Certificate {
	t.Helper()

	key := newKey(t)
	return sign(t, client(t, subject), a.Certificate, &key.PublicKey, a.key)
}

// SelfSigned makes a client certificate signed by its own key.
func SelfSigned(t testing.TB, name string) *x509.Certificate {
	t.Helper()

	key := newKey(t)
	template := client(t, Subject{CommonName: name})
	return sign(t, template, template, &key.PublicKey, key)
}

// PEM is the certificate as an administrator pastes it.
func PEM(cert *x509.Certificate) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
}

// Header is the certificate as nginx's $ssl_client_escaped_cert passes it.
func Header(cert *x509.Certificate) string {
	return url.PathEscape(PEM(cert))
}

func client(t testing.TB, subject Subject) *x509.Certificate {
	notAfter := subject.NotAfter
	if notAfter.IsZero() {
		notAfter = time.Now().Add(24 * time.Hour)
	}

	template := &x509.Certificate{
		SerialNumber:   serial(t),
		Subject:        pkix.Name{CommonName: subject.CommonName, Organization: []string{"Example"}},
		NotBefore:      time.Now().Add(-time.Hour),
		NotAfter:       notAfter,
		KeyUsage:       x509.KeyUsageDigitalSignature,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		DNSNames:       subject.DNS,
		EmailAddresses: subject.Emails,
	}
	for _, raw := range subject.URIs {
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatal(err)
		}
		template.URIs = append(template.URIs, parsed)
	}
	for _, ip := range subject.IPs {
		template.IPAddresses = append(template.IPAddresses, net.ParseIP(ip))
	}

	return template
}

func sign(t testing.TB, template, parent *x509.Certificate, public *ecdsa.PublicKey, signer *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()

	der, err := x509.CreateCertificate(rand.Reader, template, parent, public, signer)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func newKey(t testing.TB) *ecdsa.PrivateKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func serial(t testing.TB) *big.Int {
	t.Helper()

	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		t.Fatal(err)
	}
	return n
}
