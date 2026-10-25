package oidc

import (
	"context"
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

func metadataOf(app *model.Application) ClientMetadata {
	kind := "web"
	if app.Type == model.AppNative {
		kind = "native"
	}

	return ClientMetadata{
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
	}
}
