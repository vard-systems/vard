package oidc

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"slices"
	"strings"

	"vard/internal/model"
)

// Dynamic client registration (RFC 7591). Registering is not open: it takes
// an initial access token, which is an admin API token with
// applications.write — the same an operator's tooling uses to manage
// applications — so whoever registers clients is an administrator's service.
// A registered client is an ordinary application, held to the same rules as
// one made in the panel.

const PathRegistration = "/oauth2/register"

// Registration errors (RFC 7591 §3.2.2).
const (
	ErrInvalidRedirectURI    = "invalid_redirect_uri"
	ErrInvalidClientMetadata = "invalid_client_metadata"
)

// ClientMetadata is what a client registers with (RFC 7591 §2, with the
// logout URIs of OpenID Connect's logout specifications).
type ClientMetadata struct {
	RedirectURIs            []string `json:"redirect_uris"`
	PostLogoutRedirectURIs  []string `json:"post_logout_redirect_uris,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method,omitempty"`
	GrantTypes              []string `json:"grant_types,omitempty"`
	ResponseTypes           []string `json:"response_types,omitempty"`
	ApplicationType         string   `json:"application_type,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	ClientURI               string   `json:"client_uri,omitempty"`
	LogoURI                 string   `json:"logo_uri,omitempty"`
	TosURI                  string   `json:"tos_uri,omitempty"`
	PolicyURI               string   `json:"policy_uri,omitempty"`
	Scope                   string   `json:"scope,omitempty"`
	BackchannelLogoutURI    string   `json:"backchannel_logout_uri,omitempty"`
	FrontchannelLogoutURI   string   `json:"frontchannel_logout_uri,omitempty"`

	RequirePushedAuthorizationRequests bool `json:"require_pushed_authorization_requests,omitempty"`

	// RFC 8705: the one subject a tls_client_auth certificate carries, the
	// certificates of a self_signed_tls_client_auth client (as x5c in its
	// jwks), and whether its access tokens are bound to the certificate.
	TLSClientAuthSubjectDN                string      `json:"tls_client_auth_subject_dn,omitempty"`
	TLSClientAuthSANDNS                   string      `json:"tls_client_auth_san_dns,omitempty"`
	TLSClientAuthSANURI                   string      `json:"tls_client_auth_san_uri,omitempty"`
	TLSClientAuthSANIP                    string      `json:"tls_client_auth_san_ip,omitempty"`
	TLSClientAuthSANEmail                 string      `json:"tls_client_auth_san_email,omitempty"`
	JWKS                                  *ClientKeys `json:"jwks,omitempty"`
	JWKSURI                               string      `json:"jwks_uri,omitempty"`
	TLSClientCertificateBoundAccessTokens bool        `json:"tls_client_certificate_bound_access_tokens,omitempty"`
}

// ClientKeys is a client's JSON Web Key Set as far as it is read here: the
// certificate of each key (x5c), which a self-signed client registers.
type ClientKeys struct {
	Keys []ClientKey `json:"keys"`
}

type ClientKey struct {
	X5C []string `json:"x5c,omitempty"`
}

// RegisteredClient is the answer: the client's id, its secret when it has
// one (shown this once), and its metadata as the server now holds it.
type RegisteredClient struct {
	ClientID              string `json:"client_id"`
	ClientSecret          string `json:"client_secret,omitempty"`
	ClientIDIssuedAt      int64  `json:"client_id_issued_at"`
	ClientSecretExpiresAt int64  `json:"client_secret_expires_at"`
	ClientMetadata
}

// RegisterClient makes an application from a client's metadata. The token
// is the initial access token; who registered is in the activity log.
func (s *Service) RegisterClient(ctx context.Context, token string, m ClientMetadata) (*RegisteredClient, error) {
	caller, err := s.AdminCaller(ctx, token)
	if errors.Is(err, ErrTokenRefused) {
		return nil, oauthError(ErrInvalidToken, "the initial access token is not valid")
	}
	if err != nil {
		return nil, err
	}
	if !slices.Contains(caller.Permissions, string(model.PermApplicationsWrite)) {
		return nil, oauthError(ErrInsufficientScope, "the initial access token needs "+string(model.PermApplicationsWrite))
	}

	app, err := applicationFrom(m)
	if err != nil {
		return nil, err
	}
	if (app.TokenAuthMethod.UsesCertificate() || app.CertificateBoundTokens) && !s.MutualTLS() {
		return nil, oauthError(ErrInvalidClientMetadata, "mutual TLS is not offered by this server")
	}
	if app.TokenAuthMethod == model.AuthTLSClient && s.clientCAs == nil {
		return nil, oauthError(ErrInvalidClientMetadata, "tls_client_auth is not offered by this server; use self_signed_tls_client_auth")
	}

	if app.ClientID, err = model.NewClientID(); err != nil {
		return nil, err
	}

	var secret string
	if app.HasSecret() {
		if secret, err = app.IssueSecret(s.now()); err != nil {
			return nil, err
		}
	}

	if err := s.store.CreateApplication(ctx, app); err != nil {
		return nil, err
	}

	entry := model.AuditLog{
		ActorEmail: caller.Application.Name,
		Action:     "application.registered",
		TargetType: "application",
		TargetID:   app.ID.String(),
		Metadata:   map[string]any{"by": caller.Application.ClientID},
	}
	if err := s.store.WriteAudit(ctx, &entry); err != nil {
		s.log.Error("writing the activity log failed", "error", err, "action", entry.Action)
	}

	return &RegisteredClient{
		ClientID:         app.ClientID,
		ClientSecret:     secret,
		ClientIDIssuedAt: app.CreatedAt.Unix(),
		ClientMetadata:   metadataOf(app),
	}, nil
}

// applicationFrom turns registered metadata into an application, refusing
// what the panel would refuse.
func applicationFrom(m ClientMetadata) (*model.Application, error) {
	grants := m.GrantTypes
	if len(grants) == 0 {
		grants = []string{model.GrantAuthorizationCode}
	}

	for _, kind := range m.ResponseTypes {
		if kind != "code" {
			return nil, oauthError(ErrInvalidClientMetadata, "response_types: only code is offered")
		}
	}

	kind := model.AppWeb
	switch {
	case m.ApplicationType == "native":
		kind = model.AppNative
	case m.ApplicationType != "" && m.ApplicationType != "web":
		return nil, oauthError(ErrInvalidClientMetadata, "application_type must be web or native")
	case len(grants) == 1 && grants[0] == model.GrantClientCredentials:
		kind = model.AppM2M
	case m.TokenEndpointAuthMethod == string(model.AuthNone):
		kind = model.AppSPA
	}

	kindOfSubject, subject, err := tlsSubjectOf(m)
	if err != nil {
		return nil, err
	}
	certificates, err := certificatesOf(m)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(m.ClientName)
	if name == "" {
		name = "Registered client"
	}

	app := &model.Application{
		Name:                   model.Truncate(name, 100),
		Type:                   kind,
		LogoURL:                m.LogoURI,
		WebsiteURL:             m.ClientURI,
		TermsURL:               m.TosURI,
		PrivacyURL:             m.PolicyURI,
		TokenAuthMethod:        model.AuthMethod(m.TokenEndpointAuthMethod),
		GrantTypes:             grants,
		RedirectURIs:           m.RedirectURIs,
		PostLogoutRedirectURIs: m.PostLogoutRedirectURIs,
		Scopes:                 strings.Fields(m.Scope),
		BackchannelLogoutURI:   m.BackchannelLogoutURI,
		FrontchannelLogoutURI:  m.FrontchannelLogoutURI,
		TLSSubjectKind:         kindOfSubject,
		TLSSubject:             subject,
		TLSCertificates:        certificates,
		CertificateBoundTokens: m.TLSClientCertificateBoundAccessTokens,
		RequirePushedRequests:  m.RequirePushedAuthorizationRequests,
		RequirePKCE:            true,
		AssertRoles:            true,
		IsEnabled:              true,
		AccessTokenLifetime:    model.DefaultAccessTokenLifetime,
		IDTokenLifetime:        model.DefaultIDTokenLifetime,
		RefreshTokenLifetime:   model.DefaultRefreshTokenLifetime,
	}
	if len(app.Scopes) == 0 && kind != model.AppM2M {
		app.Scopes = []string{model.ScopeOpenID}
	}

	app.Normalize()
	if err := app.Validate(); err != nil {
		code := ErrInvalidClientMetadata
		if strings.Contains(err.Error(), "redirect_uris") {
			code = ErrInvalidRedirectURI
		}
		return nil, oauthError(code, err.Error())
	}

	return app, nil
}

// tlsSubjectOf is the one tls_client_auth subject the metadata names.
func tlsSubjectOf(m ClientMetadata) (model.TLSSubjectKind, string, error) {
	named := []struct {
		kind  model.TLSSubjectKind
		value string
	}{
		{model.TLSSubjectDN, m.TLSClientAuthSubjectDN},
		{model.TLSSubjectDNS, m.TLSClientAuthSANDNS},
		{model.TLSSubjectURI, m.TLSClientAuthSANURI},
		{model.TLSSubjectIP, m.TLSClientAuthSANIP},
		{model.TLSSubjectEmail, m.TLSClientAuthSANEmail},
	}

	var kind model.TLSSubjectKind
	var value string
	for _, entry := range named {
		if entry.value == "" {
			continue
		}
		if kind != "" {
			return "", "", oauthError(ErrInvalidClientMetadata, "name one tls_client_auth subject, not several (RFC 8705 §2.1.2)")
		}
		kind, value = entry.kind, entry.value
	}

	return kind, value, nil
}

// certificatesOf is the PEM of the certificates in the metadata's jwks: each
// key's own, the first of its x5c.
func certificatesOf(m ClientMetadata) ([]string, error) {
	if m.JWKSURI != "" {
		return nil, oauthError(ErrInvalidClientMetadata, "jwks_uri is not offered; register the certificates in jwks")
	}
	if m.JWKS == nil {
		return []string{}, nil
	}

	out := []string{}
	for _, key := range m.JWKS.Keys {
		if len(key.X5C) == 0 {
			continue
		}
		der, err := base64.StdEncoding.DecodeString(key.X5C[0])
		if err != nil {
			return nil, oauthError(ErrInvalidClientMetadata, "jwks: an x5c is not base64")
		}
		out = append(out, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})))
	}

	return out, nil
}

func metadataOf(app *model.Application) ClientMetadata {
	kind := "web"
	if app.Type == model.AppNative {
		kind = "native"
	}

	m := ClientMetadata{
		RedirectURIs:            app.RedirectURIs,
		PostLogoutRedirectURIs:  app.PostLogoutRedirectURIs,
		TokenEndpointAuthMethod: string(app.TokenAuthMethod),
		GrantTypes:              app.GrantTypes,
		ResponseTypes:           app.ResponseTypes(),
		ApplicationType:         kind,
		ClientName:              app.Name,
		ClientURI:               app.WebsiteURL,
		LogoURI:                 app.LogoURL,
		TosURI:                  app.TermsURL,
		PolicyURI:               app.PrivacyURL,
		Scope:                   strings.Join(app.Scopes, " "),
		BackchannelLogoutURI:    app.BackchannelLogoutURI,
		FrontchannelLogoutURI:   app.FrontchannelLogoutURI,

		RequirePushedAuthorizationRequests:    app.RequirePushedRequests,
		TLSClientCertificateBoundAccessTokens: app.CertificateBoundTokens,
	}

	switch app.TLSSubjectKind {
	case model.TLSSubjectDN:
		m.TLSClientAuthSubjectDN = app.TLSSubject
	case model.TLSSubjectDNS:
		m.TLSClientAuthSANDNS = app.TLSSubject
	case model.TLSSubjectURI:
		m.TLSClientAuthSANURI = app.TLSSubject
	case model.TLSSubjectIP:
		m.TLSClientAuthSANIP = app.TLSSubject
	case model.TLSSubjectEmail:
		m.TLSClientAuthSANEmail = app.TLSSubject
	}

	if len(app.TLSCertificates) > 0 {
		m.JWKS = &ClientKeys{Keys: []ClientKey{}}
		for _, text := range app.TLSCertificates {
			if block, _ := pem.Decode([]byte(text)); block != nil {
				m.JWKS.Keys = append(m.JWKS.Keys, ClientKey{X5C: []string{base64.StdEncoding.EncodeToString(block.Bytes)}})
			}
		}
	}

	return m
}
