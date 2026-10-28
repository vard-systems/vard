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
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"

	"vard/internal/brand"
	"vard/internal/config"
	"vard/internal/jose"
	"vard/internal/mail"
	"vard/internal/model"
	"vard/internal/passkey"
	"vard/internal/store"
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
	PageLogout    = "/logout"
)

type Service struct {
	store      *store.Store
	keys       *keySet
	mail       mail.Sender
	log        *slog.Logger
	issuer     string
	accountURL string

	// mtlsURL is the mutual TLS alias of the provider's endpoints, "" when
	// mutual TLS is off; clientCAs are what a tls_client_auth certificate
	// must chain to, nil when only self-signed ones are taken.
	mtlsURL   string
	clientCAs *x509.CertPool

	sealer   *jose.Sealer
	social   *http.Client
	ssoCache *ssoCache

	backchannel *backchannelLogout
	webauthn    *webauthn.WebAuthn

	// challenge is asked of sign-ins that have failed often.
	challenge ChallengeHook

	// lookupTXT reads a domain's TXT records, for proving SSO domains.
	lookupTXT func(ctx context.Context, name string) ([]string, error)

	now func() time.Time
}

func New(ctx context.Context, cfg config.Config, st *store.Store, sealer *jose.Sealer, mailer mail.Sender, log *slog.Logger) (*Service, error) {
	keys, err := loadKeys(ctx, st, sealer, cfg.KeyRotation, log)
	if err != nil {
		return nil, err
	}

	passkeys, err := passkey.RelyingParty(cfg.AccountURL)
	if err != nil {
		return nil, err
	}

	clientCAs, err := loadClientCAs(cfg.MTLSCAFile)
	if err != nil {
		return nil, err
	}

	s := &Service{
		store:      st,
		keys:       keys,
		mail:       mailer,
		log:        log,
		issuer:     cfg.Issuer,
		accountURL: cfg.AccountURL,
		mtlsURL:    cfg.MTLSURL,
		clientCAs:  clientCAs,
		sealer:     sealer,
		social:     newFederationClient(socialTimeout, onLoopback(cfg.Issuer)),
		ssoCache:   newSSOCache(),
		backchannel: &backchannelLogout{
			client:   newBackchannelClient(onLoopback(cfg.Issuer)),
			stopping: make(chan struct{}),
		},
		webauthn:  passkeys,
		lookupTXT: lookupTXT,
		challenge: noChallenge{},
		now:       time.Now,
	}

	// Every path that ends sessions goes through the store; it finds the
	// applications to tell and hands them here after the commit.
	st.OnLogout(s.TellApplications)

	return s, nil
}

// loadClientCAs reads the authorities tls_client_auth certificates chain to;
// a file named but unreadable or empty stops the server rather than quietly
// refusing every such client.
func loadClientCAs(file string) (*x509.CertPool, error) {
	if file == "" {
		return nil, nil
	}

	text, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("mutual TLS CA file: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(text) {
		return nil, fmt.Errorf("mutual TLS CA file %s holds no PEM certificate", file)
	}

	return pool, nil
}

// MutualTLS says clients can reach the provider with a TLS certificate.
func (s *Service) MutualTLS() bool {
	return s.mtlsURL != ""
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

// Issuer is the server's issuer, the origin clients address it at.
func (s *Service) Issuer() string {
	return s.issuer
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

	// Back-Channel Logout 1.0: applications with a backchannel_logout_uri
	// are told when a session ends, and the logout token always has a sid.
	BackchannelLogoutSupported        bool `json:"backchannel_logout_supported"`
	BackchannelLogoutSessionSupported bool `json:"backchannel_logout_session_supported"`

	// Front-Channel Logout 1.0: applications with a frontchannel_logout_uri
	// are loaded in the signing-out browser, always with iss and sid.
	FrontchannelLogoutSupported        bool `json:"frontchannel_logout_supported"`
	FrontchannelLogoutSessionSupported bool `json:"frontchannel_logout_session_supported"`

	// RFC 9126: requests may be pushed first; each application says whether
	// it must, so the server as a whole does not require it.
	PushedAuthorizationRequestEndpoint string `json:"pushed_authorization_request_endpoint"`
	DeviceAuthorizationEndpoint        string `json:"device_authorization_endpoint"`
	// RegistrationEndpoint takes clients' metadata (RFC 7591), with an
	// initial access token.
	RegistrationEndpoint string `json:"registration_endpoint"`

	// RFC 9449: proofs are accepted signed with these.
	DPoPSigningAlgValuesSupported      []string `json:"dpop_signing_alg_values_supported"`
	RequirePushedAuthorizationRequests bool     `json:"require_pushed_authorization_requests"`

	// The organisation's terms and privacy links (op_tos_uri, op_policy_uri),
	// omitted when unset.
	OpTosURI    string `json:"op_tos_uri,omitempty"`
	OpPolicyURI string `json:"op_policy_uri,omitempty"`

	// RFC 8705, when mutual TLS is on: the endpoints a client reaches with
	// its certificate, and access tokens bound to it.
	TLSClientCertificateBoundAccessTokens bool                 `json:"tls_client_certificate_bound_access_tokens"`
	MTLSEndpointAliases                   *MTLSEndpointAliases `json:"mtls_endpoint_aliases,omitempty"`

	ClaimsParameterSupported     bool `json:"claims_parameter_supported"`
	RequestParameterSupported    bool `json:"request_parameter_supported"`
	RequestURIParameterSupported bool `json:"request_uri_parameter_supported"`
}

// MTLSEndpointAliases are the endpoints a client calls over mutual TLS (RFC
// 8705 §5): the same paths on the mutual TLS host.
type MTLSEndpointAliases struct {
	TokenEndpoint                      string `json:"token_endpoint"`
	RevocationEndpoint                 string `json:"revocation_endpoint"`
	IntrospectionEndpoint              string `json:"introspection_endpoint"`
	UserInfoEndpoint                   string `json:"userinfo_endpoint"`
	PushedAuthorizationRequestEndpoint string `json:"pushed_authorization_request_endpoint"`
	DeviceAuthorizationEndpoint        string `json:"device_authorization_endpoint"`
}

// MTLSPaths are the paths the mutual TLS host serves; its proxy passes no
// other on (deploy/nginx).
var MTLSPaths = []string{PathToken, PathRevoke, PathIntrospect, PathUserInfo, PathPushedAuthorization, PathDeviceAuthorization}

// Discovery describes the provider from the model, so it cannot claim anything
// the server does not offer. Terms and privacy links come from the database; if
// it is unreachable they are left out and the rest is still served.
func (s *Service) Discovery(ctx context.Context) Discovery {
	clientMethods := []string{string(model.AuthClientSecretBasic), string(model.AuthClientSecretPost)}

	var aliases *MTLSEndpointAliases
	if s.MutualTLS() {
		aliases = &MTLSEndpointAliases{
			TokenEndpoint:                      s.mtlsURL + PathToken,
			RevocationEndpoint:                 s.mtlsURL + PathRevoke,
			IntrospectionEndpoint:              s.mtlsURL + PathIntrospect,
			UserInfoEndpoint:                   s.mtlsURL + PathUserInfo,
			PushedAuthorizationRequestEndpoint: s.mtlsURL + PathPushedAuthorization,
			DeviceAuthorizationEndpoint:        s.mtlsURL + PathDeviceAuthorization,
		}
		if s.clientCAs != nil {
			clientMethods = append(clientMethods, string(model.AuthTLSClient))
		}
		clientMethods = append(clientMethods, string(model.AuthSelfSignedTLSClient))
	}

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
		TokenEndpointAuthMethodsSupported:          append(clientMethods, string(model.AuthNone)),
		RevocationEndpointAuthMethodsSupported:     append(append([]string{}, clientMethods...), string(model.AuthNone)),
		IntrospectionEndpointAuthMethodsSupported:  clientMethods,
		CodeChallengeMethodsSupported:              []string{model.PKCES256},
		PromptValuesSupported:                      []string{"none", "login", "consent"},
		AuthorizationResponseIssParameterSupported: true,
		BackchannelLogoutSupported:                 true,
		BackchannelLogoutSessionSupported:          true,
		FrontchannelLogoutSupported:                true,
		FrontchannelLogoutSessionSupported:         true,
		PushedAuthorizationRequestEndpoint:         s.issuer + PathPushedAuthorization,
		DeviceAuthorizationEndpoint:                s.issuer + PathDeviceAuthorization,
		RegistrationEndpoint:                       s.issuer + PathRegistration,
		DPoPSigningAlgValuesSupported:              dpopAlgorithms,
		TLSClientCertificateBoundAccessTokens:      s.MutualTLS(),
		MTLSEndpointAliases:                        aliases,
		OpTosURI:                                   tos,
		OpPolicyURI:                                policy,
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
	// Device is the browser's known-device cookie, "" for none.
	Device string
	// Challenge is the sign-in form's answer to the challenge hook.
	Challenge string
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
