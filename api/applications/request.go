package applications

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/api/kit/query"
	"vard/core/model"
	"vard/core/store"
)

const targetType = "application"

// maxPageSize caps how many applications one request can ask for. The panel
// asks for every application at once to offer them as choices.
const maxPageSize = 500

// maxURIs caps how many redirect URIs of each kind an application may have.
const maxURIs = 20

// applicationRequest is the body of the create and update endpoints, named
// after RFC 7591 client metadata. Type is read on create only. A lifetime of
// zero takes the default.
type applicationRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Description string `json:"description" validate:"max=255"`
	Type        string `json:"type"`

	LogoURL    string `json:"logo_url" validate:"max=512"`
	WebsiteURL string `json:"website_url" validate:"max=512"`
	PrivacyURL string `json:"privacy_url" validate:"max=512"`
	TermsURL   string `json:"terms_url" validate:"max=512"`

	TokenAuthMethod        string   `json:"token_auth_method"`
	GrantTypes             []string `json:"grant_types"`
	RedirectURIs           []string `json:"redirect_uris" validate:"dive,max=512"`
	PostLogoutRedirectURIs []string `json:"post_logout_redirect_uris" validate:"dive,max=512"`
	BackchannelLogoutURI   string   `json:"backchannel_logout_uri" validate:"max=512"`
	FrontchannelLogoutURI  string   `json:"frontchannel_logout_uri" validate:"max=512"`
	// AllowDevelopmentRedirects lets a native app register exp:// addresses.
	AllowDevelopmentRedirects *bool    `json:"allow_development_redirects"`
	Scopes                    []string `json:"scopes"`
	RequirePKCE               *bool    `json:"require_pkce"`
	RequirePushedRequests     *bool    `json:"require_pushed_authorization_requests"`

	AccessTokenLifetime  int `json:"access_token_lifetime"`
	IDTokenLifetime      int `json:"id_token_lifetime"`
	RefreshTokenLifetime int `json:"refresh_token_lifetime"`

	// The flags may be left out: a new application is then enabled, requires
	// PKCE and puts roles in its tokens; an existing one keeps its settings.
	AssertRoles           *bool `json:"assert_roles"`
	RequireRoleAssignment *bool `json:"require_role_assignment"`
	IsEnabled             *bool `json:"is_enabled"`
	// AllowRegistration may be left out too: a new application then offers
	// registration on its sign-in page.
	AllowRegistration *bool `json:"allow_registration"`

	// LoginFlowID: "" returns to the default flow, omitted keeps the current
	// one.
	LoginFlowID *string `json:"login_flow_id"`
}

func (r *applicationRequest) clean() {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	r.Type = strings.TrimSpace(r.Type)
	r.LogoURL = strings.TrimSpace(r.LogoURL)
	r.WebsiteURL = strings.TrimSpace(r.WebsiteURL)
	r.PrivacyURL = strings.TrimSpace(r.PrivacyURL)
	r.TermsURL = strings.TrimSpace(r.TermsURL)
	r.TokenAuthMethod = strings.TrimSpace(r.TokenAuthMethod)
	r.BackchannelLogoutURI = strings.TrimSpace(r.BackchannelLogoutURI)
	r.FrontchannelLogoutURI = strings.TrimSpace(r.FrontchannelLogoutURI)

	if r.AccessTokenLifetime == 0 {
		r.AccessTokenLifetime = model.DefaultAccessTokenLifetime
	}
	if r.IDTokenLifetime == 0 {
		r.IDTokenLifetime = model.DefaultIDTokenLifetime
	}
	if r.RefreshTokenLifetime == 0 {
		r.RefreshTokenLifetime = model.DefaultRefreshTokenLifetime
	}
}

// authorizeRequest is the body of the endpoint that authorises an application
// for an API: the ids of the API's scopes it may ask for.
type authorizeRequest struct {
	Scopes []uuid.UUID `json:"scopes"`
}

// previewRequest is a token request to evaluate: the user (nil for the
// application itself), the API and the scope parameter.
type previewRequest struct {
	UserID   *uuid.UUID `json:"user_id"`
	Audience string     `json:"audience"`
	Scope    string     `json:"scope"`
}

func listQuery(c *gin.Context) store.ApplicationQuery {
	q := store.ApplicationQuery{
		Search:  c.Query("search"),
		Limit:   query.Int(c, "limit", 50, maxPageSize),
		Offset:  query.Int(c, "offset", 0, query.MaxOffset),
		Enabled: query.Bool(c, "enabled"),
	}

	if kind := model.ApplicationType(c.Query("type")); kind.Valid() {
		q.Type = kind
	}

	return q
}
