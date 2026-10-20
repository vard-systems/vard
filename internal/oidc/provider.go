// Package oidc is the OAuth 2.0 authorization server and OpenID Connect
// provider, and everything users do to sign themselves in: passwords,
// registration, resets, email verification, social sign-in and SSO, and their
// own account. Administrators sign in through internal/auth instead.
//
// It knows nothing of HTTP: handlers in internal/api call it, and what a token
// carries is decided by model.EvaluateToken alone.
package oidc

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"loginer/internal/brand"
	"loginer/internal/config"
	"loginer/internal/jose"
	"loginer/internal/mail"
	"loginer/internal/model"
	"loginer/internal/store"
)

const (
	PathDiscovery  = "/.well-known/openid-configuration"
	PathJWKS       = "/.well-known/jwks.json"
	PathAuthorize  = "/oauth2/authorize"
	PathToken      = "/oauth2/token"
	PathUserInfo   = "/oauth2/userinfo"
	PathLogout     = "/oauth2/logout"
	PathRevoke     = "/oauth2/revoke"
	PathIntrospect = "/oauth2/introspect"
)

const (
	PageLogin     = "/login"
	PageReset     = "/reset-password"
	PageVerify    = "/verify-email"
	PageError     = "/error"
	PageLoggedOut = "/logged-out"
)

type Service struct {
	store      *store.Store
	keys       *keySet
	mail       mail.Sender
	log        *slog.Logger
	issuer     string
	accountURL string

	sealer   *jose.Sealer
	social   *http.Client
	ssoCache *ssoCache

	now func() time.Time
}

func New(ctx context.Context, cfg config.Config, st *store.Store, sealer *jose.Sealer, mailer mail.Sender, log *slog.Logger) (*Service, error) {
	keys, err := loadKeys(ctx, st, sealer, cfg.KeyRotation, log)
	if err != nil {
		return nil, err
	}

	return &Service{
		store:      st,
		keys:       keys,
		mail:       mailer,
		log:        log,
		issuer:     cfg.Issuer,
		accountURL: cfg.AccountURL,
		sealer:     sealer,
		social:     newFederationClient(socialTimeout, onLoopback(cfg.Issuer)),
		ssoCache:   newSSOCache(),
		now:        time.Now,
	}, nil
}

func (s *Service) MaintainKeys(ctx context.Context) {
	ticker := time.NewTicker(keyMaintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.keys.maintain(ctx); err != nil && ctx.Err() == nil {
				s.log.Error("maintaining signing keys failed", "error", err)
			}
		}
	}
}

func (s *Service) SigningKeys() []KeyInfo {
	return s.keys.describe()
}

func (s *Service) RotateKeys(ctx context.Context, immediate, revoke bool) error {
	return s.keys.Rotate(ctx, immediate, revoke)
}

// Discovery is the OpenID Connect Discovery 1.0 document.
type Discovery struct {
	Issuer                                     string   `json:"issuer"`
	AuthorizationEndpoint                      string   `json:"authorization_endpoint"`
	TokenEndpoint                              string   `json:"token_endpoint"`
	UserInfoEndpoint                           string   `json:"userinfo_endpoint"`
	EndSessionEndpoint                         string   `json:"end_session_endpoint"`
	RevocationEndpoint                         string   `json:"revocation_endpoint"`
	IntrospectionEndpoint                      string   `json:"introspection_endpoint"`
	JWKSURI                                    string   `json:"jwks_uri"`
	ResponseTypesSupported                     []string `json:"response_types_supported"`
	ResponseModesSupported                     []string `json:"response_modes_supported"`
	GrantTypesSupported                        []string `json:"grant_types_supported"`
	SubjectTypesSupported                      []string `json:"subject_types_supported"`
	IDTokenSigningAlgValuesSupported           []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                            []string `json:"scopes_supported"`
	ClaimsSupported                            []string `json:"claims_supported"`
	TokenEndpointAuthMethodsSupported          []string `json:"token_endpoint_auth_methods_supported"`
	RevocationEndpointAuthMethodsSupported     []string `json:"revocation_endpoint_auth_methods_supported"`
	IntrospectionEndpointAuthMethodsSupported  []string `json:"introspection_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported              []string `json:"code_challenge_methods_supported"`
	PromptValuesSupported                      []string `json:"prompt_values_supported"`
	AuthorizationResponseIssParameterSupported bool     `json:"authorization_response_iss_parameter_supported"`

	// The organisation's terms and privacy links (op_tos_uri, op_policy_uri),
	// omitted when unset.
	OpTosURI    string `json:"op_tos_uri,omitempty"`
	OpPolicyURI string `json:"op_policy_uri,omitempty"`

	ClaimsParameterSupported     bool `json:"claims_parameter_supported"`
	RequestParameterSupported    bool `json:"request_parameter_supported"`
	RequestURIParameterSupported bool `json:"request_uri_parameter_supported"`
}

// Discovery describes the provider from the model, so it cannot claim anything
// the server does not offer. Terms and privacy links come from the database; if
// it is unreachable they are left out and the rest is still served.
func (s *Service) Discovery(ctx context.Context) Discovery {
	secretMethods := []string{string(model.AuthClientSecretBasic), string(model.AuthClientSecretPost)}

	var tos, policy string
	if organization, err := s.store.Organization(ctx); err == nil {
		tos, policy = organization.TermsURL, organization.PrivacyURL
	} else {
		s.log.Error("reading the organization for the discovery document failed", "error", err)
	}

	return Discovery{
		Issuer:                                     s.issuer,
		AuthorizationEndpoint:                      s.issuer + PathAuthorize,
		TokenEndpoint:                              s.issuer + PathToken,
		UserInfoEndpoint:                           s.issuer + PathUserInfo,
		EndSessionEndpoint:                         s.issuer + PathLogout,
		RevocationEndpoint:                         s.issuer + PathRevoke,
		IntrospectionEndpoint:                      s.issuer + PathIntrospect,
		JWKSURI:                                    s.issuer + PathJWKS,
		ResponseTypesSupported:                     []string{"code"},
		ResponseModesSupported:                     []string{"query"},
		GrantTypesSupported:                        model.GrantTypes,
		SubjectTypesSupported:                      []string{"public"},
		IDTokenSigningAlgValuesSupported:           []string{idTokenAlgorithm},
		ScopesSupported:                            model.Scopes,
		ClaimsSupported:                            claimsSupported,
		TokenEndpointAuthMethodsSupported:          append(secretMethods, string(model.AuthNone)),
		RevocationEndpointAuthMethodsSupported:     append(append([]string{}, secretMethods...), string(model.AuthNone)),
		IntrospectionEndpointAuthMethodsSupported:  secretMethods,
		CodeChallengeMethodsSupported:              []string{model.PKCES256},
		PromptValuesSupported:                      []string{"none", "login"},
		AuthorizationResponseIssParameterSupported: true,
		OpTosURI:    tos,
		OpPolicyURI: policy,
	}
}

var claimsSupported = []string{
	"iss", "sub", "aud", "exp", "iat", "auth_time", "nonce", "at_hash", "sid",
	"azp", "client_id", "scope", "jti",
	"name", "given_name", "family_name", "email", "email_verified",
	model.ClaimApplicationRoles, model.ClaimGlobalRoles,
}

// JWKS is the JSON Web Key Set.
type JWKS struct {
	Keys []jose.JWK `json:"keys"`
}

func (s *Service) JWKS(ctx context.Context) (JWKS, error) {
	keys, err := s.keys.public(ctx)
	return JWKS{Keys: keys}, err
}

type Client struct {
	IP        string
	UserAgent string
	Language  string
}

const LanguageCookie = brand.LanguageCookie

func withQuery(base string, values url.Values) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}

	query := u.Query()
	for key, vals := range values {
		for _, v := range vals {
			if v != "" {
				query.Add(key, v)
			}
		}
	}
	u.RawQuery = query.Encode()

	return u.String()
}

// errorPage is for authorization errors that cannot be redirected (unknown
// client, unregistered redirect URI), since redirecting would hand the user to
// whoever wrote the link.
func (s *Service) errorPage(code, description string) string {
	return withQuery(s.accountURL+PageError, url.Values{"error": {code}, "error_description": {description}})
}

func (s *Service) record(ctx context.Context, user *model.User, email, action string, client Client, metadata map[string]any) {
	entry := model.AuditLog{
		ActorEmail: email,
		Action:     action,
		TargetType: "user",
		IP:         client.IP,
		UserAgent:  client.UserAgent,
		Metadata:   metadata,
	}
	if user != nil {
		entry.TargetID = user.ID.String()
	}

	if err := s.store.WriteAudit(ctx, &entry); err != nil {
		s.log.Error("writing the activity log failed", "error", err, "action", action)
	}
}
