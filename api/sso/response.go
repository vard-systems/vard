package sso

import (
	"crypto/x509"
	"encoding/base64"
	"strings"
	"time"

	"github.com/crewjam/saml"

	"vard/core/model"
	"vard/core/oidc"
)

// connectionResponse is a connection as the panel shows it: settings without
// secrets, user count, provider setup details and parsed SAML metadata.
type connectionResponse struct {
	model.SSOConnection

	HasClientSecret bool  `json:"has_client_secret"`
	Users           int64 `json:"users"`

	ServiceProvider  serviceProvider   `json:"service_provider"`
	IdentityProvider *identityProvider `json:"identity_provider,omitempty"`
}

// serviceProvider is what the identity provider is configured with: the OIDC
// redirect URI, or the SAML ACS, entity ID, metadata and certificate.
type serviceProvider struct {
	CallbackURL string `json:"callback_url,omitempty"`
	ACSURL      string `json:"acs_url,omitempty"`
	EntityID    string `json:"entity_id,omitempty"`
	MetadataURL string `json:"metadata_url,omitempty"`
	Certificate string `json:"certificate,omitempty"`
}

// identityProvider is what SAML metadata says, including when its certificate
// expires (the usual cause of a sudden outage).
type identityProvider struct {
	EntityID           string     `json:"entity_id"`
	SSOURL             string     `json:"sso_url"`
	Certificates       int        `json:"certificates"`
	CertificateExpires *time.Time `json:"certificate_expires,omitempty"`
}

type listResponse struct {
	Connections []connectionResponse `json:"connections"`
}

type response struct {
	Connection connectionResponse `json:"connection"`
}

// testResponse is what trying a provider found.
type testResponse struct {
	// For OpenID Connect, the endpoints its discovery document gave.
	Issuer                string `json:"issuer,omitempty"`
	AuthorizationEndpoint string `json:"authorization_endpoint,omitempty"`
	TokenEndpoint         string `json:"token_endpoint,omitempty"`
	JWKSURI               string `json:"jwks_uri,omitempty"`

	// UnsupportedScopes are requested scopes the provider does not list; some
	// providers (Keycloak) refuse the whole sign-in over one.
	UnsupportedScopes []string `json:"unsupported_scopes,omitempty"`

	// For SAML, what its metadata said, and the metadata itself when it was
	// fetched, so the panel can keep it.
	IdentityProvider *identityProvider `json:"identity_provider,omitempty"`
	Metadata         string            `json:"metadata,omitempty"`
}

func newConnectionResponse(connection model.SSOConnection, users int64, issuer string) connectionResponse {
	out := connectionResponse{
		SSOConnection:   connection,
		HasClientSecret: len(connection.ClientSecret) > 0,
		Users:           users,
	}

	switch connection.Protocol {
	case model.SSOProtocolOIDC:
		out.ServiceProvider = serviceProvider{CallbackURL: connection.CallbackURL(issuer)}
	case model.SSOProtocolSAML:
		out.ServiceProvider = serviceProvider{
			ACSURL:      connection.ACSURL(issuer),
			EntityID:    connection.EntityID(issuer),
			MetadataURL: connection.MetadataURLFor(issuer),
			Certificate: connection.SPCertificate,
		}
		if descriptor, err := oidc.ParseSAMLMetadata([]byte(connection.Metadata)); err == nil {
			out.IdentityProvider = describeIdentityProvider(descriptor)
		}
	}

	return out
}

func describeIdentityProvider(descriptor *saml.EntityDescriptor) *identityProvider {
	idp := descriptor.IDPSSODescriptors[0]
	out := &identityProvider{EntityID: descriptor.EntityID}

	for _, endpoint := range idp.SingleSignOnServices {
		if endpoint.Binding == saml.HTTPRedirectBinding {
			out.SSOURL = endpoint.Location
		}
	}

	for _, key := range idp.KeyDescriptors {
		if key.Use != "" && key.Use != "signing" {
			continue
		}
		for _, certificate := range key.KeyInfo.X509Data.X509Certificates {
			out.Certificates++
			if expires, ok := certificateExpiry(certificate.Data); ok &&
				(out.CertificateExpires == nil || expires.Before(*out.CertificateExpires)) {
				out.CertificateExpires = &expires
			}
		}
	}

	return out
}

func certificateExpiry(data string) (time.Time, bool) {
	der, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(data), ""))
	if err != nil {
		return time.Time{}, false
	}

	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		return time.Time{}, false
	}

	return certificate.NotAfter, true
}

// domainProof is one of a connection's domains: whether it is proved, how,
// and the TXT record that proves it.
type domainProof struct {
	Domain      string     `json:"domain"`
	Proved      bool       `json:"proved"`
	Method      string     `json:"method"`
	RecordName  string     `json:"record_name"`
	RecordValue string     `json:"record_value"`
	VerifiedAt  *time.Time `json:"verified_at"`
	CheckedAt   *time.Time `json:"checked_at"`
}

func newDomainProof(v model.SSODomainVerification) domainProof {
	return domainProof{
		Domain: v.Domain, Proved: v.Proved(), Method: v.Method,
		RecordName: v.RecordName(), RecordValue: v.RecordValue(),
		VerifiedAt: v.VerifiedAt, CheckedAt: v.CheckedAt,
	}
}

type domainsResponse struct {
	Domains []domainProof `json:"domains"`
}

type domainResponse struct {
	Domain domainProof `json:"domain"`
}
