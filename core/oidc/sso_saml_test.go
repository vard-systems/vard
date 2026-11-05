package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/beevik/etree"
	"github.com/crewjam/saml"
	"github.com/crewjam/saml/xmlenc"
	"github.com/google/uuid"

	"vard/core/model"
)

func TestCheckSAMLAlgorithms(t *testing.T) {
	const (
		dsig  = `xmlns:ds="http://www.w3.org/2000/09/xmldsig#"`
		xenc  = `xmlns:xenc="http://www.w3.org/2001/04/xmlenc#"`
		rsa1  = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"
		rsa2  = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
		ecd1  = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha1"
		sha1  = "http://www.w3.org/2000/09/xmldsig#sha1"
		sha2  = "http://www.w3.org/2001/04/xmlenc#sha256"
		oaep  = "http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"
		rsa15 = "http://www.w3.org/2001/04/xmlenc#rsa-1_5"
	)

	signed := func(method, digest string) string {
		return `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"><ds:Signature ` + dsig + `><ds:SignedInfo>` +
			`<ds:SignatureMethod Algorithm="` + method + `"/><ds:Reference><ds:DigestMethod Algorithm="` + digest + `"/></ds:Reference>` +
			`</ds:SignedInfo></ds:Signature></samlp:Response>`
	}
	encrypted := func(transport string) string {
		return `<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"><saml:EncryptedAssertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">` +
			`<xenc:EncryptedData ` + xenc + `><ds:KeyInfo ` + dsig + `><xenc:EncryptedKey><xenc:EncryptionMethod Algorithm="` + transport + `">` +
			`<ds:DigestMethod Algorithm="` + sha1 + `"/></xenc:EncryptionMethod></xenc:EncryptedKey></ds:KeyInfo></xenc:EncryptedData>` +
			`</saml:EncryptedAssertion></samlp:Response>`
	}

	tests := []struct {
		name, xml, refused string
	}{
		{name: "RSA-SHA256 with a SHA-256 digest", xml: signed(rsa2, sha2)},
		{name: "an RSA-SHA1 signature", xml: signed(rsa1, sha2), refused: "SHA-1"},
		{name: "an ECDSA-SHA1 signature", xml: signed(ecd1, sha2), refused: "SHA-1"},
		{name: "a SHA-1 digest under a SHA-256 signature", xml: signed(rsa2, sha1), refused: "SHA-1"},
		// OAEP's own hash defaults to SHA-1 and is not a signature.
		{name: "RSA-OAEP key transport with its SHA-1 digest", xml: encrypted(oaep)},
		{name: "RSA PKCS#1 v1.5 key transport", xml: encrypted(rsa15), refused: "PKCS#1 v1.5"},
		{name: "not XML", xml: "<<", refused: "not XML"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSAMLAlgorithms([]byte(tt.xml))
			switch {
			case tt.refused == "" && err != nil:
				t.Errorf("refused: %v", err)
			case tt.refused != "" && (err == nil || !strings.Contains(err.Error(), tt.refused)):
				t.Errorf("= %v, want a refusal saying %q", err, tt.refused)
			}
		})
	}
}

func TestBearerConfirmed(t *testing.T) {
	confirmed := func(methods ...string) *saml.Assertion {
		subject := &saml.Subject{}
		for _, method := range methods {
			subject.SubjectConfirmations = append(subject.SubjectConfirmations, saml.SubjectConfirmation{Method: method})
		}
		return &saml.Assertion{Subject: subject}
	}

	tests := []struct {
		name      string
		assertion *saml.Assertion
		want      bool
	}{
		{"bearer", confirmed(samlBearer), true},
		{"bearer among others", confirmed("urn:oasis:names:tc:SAML:2.0:cm:holder-of-key", samlBearer), true},
		{"holder of key only", confirmed("urn:oasis:names:tc:SAML:2.0:cm:holder-of-key"), false},
		{"sender vouches only", confirmed("urn:oasis:names:tc:SAML:2.0:cm:sender-vouches"), false},
		{"no confirmation", confirmed(), false},
		{"no subject", &saml.Assertion{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bearerConfirmed(tt.assertion); got != tt.want {
				t.Errorf("= %v, want %v", got, tt.want)
			}
		})
	}
}

// An assertion signs in once per connection, however it comes back.
func TestLiveSAMLAssertionsAreUsedOnce(t *testing.T) {
	s := &Service{store: testStore(t), log: slog.New(slog.NewTextHandler(io.Discard, nil)), now: time.Now}
	ctx := context.Background()

	assertion := &saml.Assertion{ID: "_assertion-1", Subject: &saml.Subject{}}
	acme := &model.SSOConnection{Base: model.Base{ID: uuid.New()}}
	globex := &model.SSOConnection{Base: model.Base{ID: uuid.New()}}

	if err := s.useSAMLAssertion(ctx, acme, assertion); err != nil {
		t.Fatalf("first use: %v", err)
	}

	var refusal *ssoRefusal
	if err := s.useSAMLAssertion(ctx, acme, assertion); !errors.As(err, &refusal) || !strings.Contains(err.Error(), "already used") {
		t.Errorf("second use = %v, want refused as already used", err)
	}

	// Another connection's provider may well number its assertions the same.
	if err := s.useSAMLAssertion(ctx, globex, assertion); err != nil {
		t.Errorf("the same ID at another connection: %v", err)
	}

	if err := s.useSAMLAssertion(ctx, acme, &saml.Assertion{Subject: &saml.Subject{}}); !errors.As(err, &refusal) {
		t.Errorf("an assertion without an ID = %v, want refused", err)
	}
}

// A signature is checked for SHA-1 inside an encrypted assertion too, once
// it is opened with the connection's key, whether the key comes inside the
// encrypted data or beside it.
func TestCheckEncryptedAlgorithms(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Now(), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	assertion := func(method string) []byte {
		return []byte(`<saml:Assertion xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">` +
			`<ds:SignedInfo><ds:SignatureMethod Algorithm="` + method + `"/></ds:SignedInfo></ds:Signature></saml:Assertion>`)
	}
	response := func(plain []byte, keyBeside bool) []byte {
		data, err := xmlenc.OAEP().Encrypt(certificate, plain, nil)
		if err != nil {
			t.Fatal(err)
		}
		encrypted := etree.NewElement("saml:EncryptedAssertion")
		encrypted.CreateAttr("xmlns:saml", "urn:oasis:names:tc:SAML:2.0:assertion")
		encrypted.AddChild(data)
		if keyBeside {
			// Some providers put the EncryptedKey beside the data, not in its KeyInfo.
			info := data.FindElement("./KeyInfo")
			wrapped := info.FindElement("./EncryptedKey")
			info.RemoveChild(wrapped)
			data.RemoveChild(info)
			encrypted.AddChild(wrapped)
		}
		document := etree.NewDocument()
		root := document.CreateElement("samlp:Response")
		root.CreateAttr("xmlns:samlp", "urn:oasis:names:tc:SAML:2.0:protocol")
		root.AddChild(encrypted)
		out, err := document.WriteToBytes()
		if err != nil {
			t.Fatal(err)
		}
		return out
	}

	const (
		rsa1 = "http://www.w3.org/2000/09/xmldsig#rsa-sha1"
		rsa2 = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
	)
	tests := []struct {
		name    string
		raw     []byte
		key     *rsa.PrivateKey
		refused string
	}{
		{name: "a SHA-256 signature inside", raw: response(assertion(rsa2), false), key: key},
		{name: "a SHA-1 signature inside", raw: response(assertion(rsa1), false), key: key, refused: "SHA-1"},
		{name: "a SHA-1 signature inside, the key beside the data", raw: response(assertion(rsa1), true), key: key, refused: "SHA-1"},
		{name: "an assertion encrypted for another key", raw: response(assertion(rsa2), false), key: other, refused: "does not open"},
		{name: "nothing encrypted", raw: assertion(rsa2), key: key},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkEncryptedAlgorithms(tt.raw, tt.key)
			switch {
			case tt.refused == "" && err != nil:
				t.Errorf("refused: %v", err)
			case tt.refused != "" && (err == nil || !strings.Contains(err.Error(), tt.refused)):
				t.Errorf("= %v, want a refusal saying %q", err, tt.refused)
			}
		})
	}
}
