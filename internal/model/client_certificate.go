package model

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
	"time"
)

// maxTLSCertificates bounds a self-signed client's registered certificates:
// enough for one in use and the next one, while it is rolled over.
const maxTLSCertificates = 4

var ErrNotACertificate = errors.New("not a PEM certificate")

// ParseCertificate reads one PEM certificate.
func ParseCertificate(text string) (*x509.Certificate, error) {
	block, rest := pem.Decode([]byte(strings.TrimSpace(text)))
	if block == nil || block.Type != "CERTIFICATE" || strings.TrimSpace(string(rest)) != "" {
		return nil, ErrNotACertificate
	}

	return x509.ParseCertificate(block.Bytes)
}

// CertificateThumbprint is the certificate's x5t#S256: the base64url SHA-256
// of its DER encoding, as an access token's cnf names it (RFC 8705 §3.1).
func CertificateThumbprint(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// MatchesCertificate says the TLS client certificate is the application's:
// for self_signed_tls_client_auth one of the certificates it registered, for
// tls_client_auth one that chains to roots and carries its registered subject.
func (a Application) MatchesCertificate(cert *x509.Certificate, roots *x509.CertPool, now time.Time) bool {
	if cert == nil {
		return false
	}

	switch a.TokenAuthMethod {
	case AuthSelfSignedTLSClient:
		// Only the key is being recognised, so the certificate's dates and
		// issuer say nothing (RFC 8705 §2.2); the exact certificate must be
		// one registered.
		presented := CertificateThumbprint(cert)
		for _, text := range a.TLSCertificates {
			if registered, err := ParseCertificate(text); err == nil && CertificateThumbprint(registered) == presented {
				return true
			}
		}
		return false

	case AuthTLSClient:
		if roots == nil {
			return false
		}
		_, err := cert.Verify(x509.VerifyOptions{
			Roots:       roots,
			CurrentTime: now,
			KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		})
		return err == nil && a.subjectMatches(cert)
	}

	return false
}

func (a Application) subjectMatches(cert *x509.Certificate) bool {
	want := a.TLSSubject
	if want == "" {
		return false
	}

	switch a.TLSSubjectKind {
	case TLSSubjectDN:
		// Compared as Go writes the DN in RFC 4514 form, not by RFC 4517
		// distinguishedNameMatch as RFC 8705 §2.1.2 asks: a DN encoded or
		// spaced otherwise does not match. A known limitation; a SAN does
		// not have it.
		return cert.Subject.String() == want
	case TLSSubjectDNS:
		return slices.ContainsFunc(cert.DNSNames, func(name string) bool { return strings.EqualFold(name, want) })
	case TLSSubjectURI:
		return slices.ContainsFunc(cert.URIs, func(uri *url.URL) bool { return uri.String() == want })
	case TLSSubjectEmail:
		return slices.ContainsFunc(cert.EmailAddresses, func(address string) bool { return strings.EqualFold(address, want) })
	case TLSSubjectIP:
		ip := net.ParseIP(want)
		return ip != nil && slices.ContainsFunc(cert.IPAddresses, func(address net.IP) bool { return address.Equal(ip) })
	}

	return false
}

func (a Application) validateCertificates() error {
	switch a.TokenAuthMethod {
	case AuthTLSClient:
		if !slices.Contains(TLSSubjectKinds, a.TLSSubjectKind) {
			return fmt.Errorf("tls_subject_kind must be one of: subject_dn, san_dns, san_uri, san_ip, san_email")
		}
		if a.TLSSubject == "" || len(a.TLSSubject) > 512 {
			return fmt.Errorf("tls_subject: tls_client_auth needs the subject the client's certificate carries")
		}
		if a.TLSSubjectKind == TLSSubjectIP && net.ParseIP(a.TLSSubject) == nil {
			return fmt.Errorf("tls_subject: %q is not an IP address", a.TLSSubject)
		}

	case AuthSelfSignedTLSClient:
		if len(a.TLSCertificates) == 0 {
			return fmt.Errorf("tls_certificates: self_signed_tls_client_auth needs the client's certificate")
		}
		if len(a.TLSCertificates) > maxTLSCertificates {
			return fmt.Errorf("tls_certificates: at most %d certificates", maxTLSCertificates)
		}
		for i, text := range a.TLSCertificates {
			if _, err := ParseCertificate(text); err != nil {
				return fmt.Errorf("tls_certificates: certificate %d is not a PEM certificate", i+1)
			}
		}
	}

	if a.TokenAuthMethod.UsesCertificate() && a.Type.Public() {
		return fmt.Errorf("token_auth_method: a public client has no credentials, so it cannot use %s", a.TokenAuthMethod)
	}

	return nil
}

func tidyCertificates(certificates []string) []string {
	out := []string{}
	for _, text := range certificates {
		text = strings.TrimSpace(text)
		if text != "" && !slices.Contains(out, text) {
			out = append(out, text)
		}
	}

	return out
}
