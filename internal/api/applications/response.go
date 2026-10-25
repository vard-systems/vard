package applications

import (
	"slices"
	"time"

	"github.com/google/uuid"

	"vard/internal/model"
	"vard/internal/store"
)

// applicationResponse is an application as the panel sees it. The secret's
// hash is never part of it.
type applicationResponse struct {
	ID          uuid.UUID             `json:"id"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Type        model.ApplicationType `json:"type"`
	LogoURL     string                `json:"logo_url"`
	WebsiteURL  string                `json:"website_url"`
	PrivacyURL  string                `json:"privacy_url"`
	TermsURL    string                `json:"terms_url"`

	ClientID         string     `json:"client_id"`
	ClientIDIssuedAt int64      `json:"client_id_issued_at"`
	HasSecret        bool       `json:"has_secret"`
	SecretHint       string     `json:"secret_hint"`
	SecretCreatedAt  *time.Time `json:"secret_created_at"`

	TokenAuthMethod        model.AuthMethod `json:"token_auth_method"`
	GrantTypes             []string         `json:"grant_types"`
	ResponseTypes          []string         `json:"response_types"`
	RedirectURIs           []string         `json:"redirect_uris"`
	PostLogoutRedirectURIs []string         `json:"post_logout_redirect_uris"`
	BackchannelLogoutURI   string           `json:"backchannel_logout_uri"`
	FrontchannelLogoutURI  string           `json:"frontchannel_logout_uri"`
	Scopes                 []string         `json:"scopes"`
	RequirePKCE            bool             `json:"require_pkce"`
	RequirePushedRequests  bool             `json:"require_pushed_authorization_requests"`

	AccessTokenLifetime  int `json:"access_token_lifetime"`
	IDTokenLifetime      int `json:"id_token_lifetime"`
	RefreshTokenLifetime int `json:"refresh_token_lifetime"`

	AssertRoles           bool `json:"assert_roles"`
	RequireRoleAssignment bool `json:"require_role_assignment"`
	IsEnabled             bool `json:"is_enabled"`
	AllowRegistration     bool `json:"allow_registration"`

	// LoginFlowID is the application's flow, null for the default.
	LoginFlowID *uuid.UUID `json:"login_flow_id"`

	// RoleCount is how many roles the application defines.
	RoleCount int64     `json:"role_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newApplicationResponse(app model.Application, roles int64) applicationResponse {
	return applicationResponse{
		ID:                     app.ID,
		Name:                   app.Name,
		Description:            app.Description,
		Type:                   app.Type,
		LogoURL:                app.LogoURL,
		WebsiteURL:             app.WebsiteURL,
		PrivacyURL:             app.PrivacyURL,
		TermsURL:               app.TermsURL,
		ClientID:               app.ClientID,
		ClientIDIssuedAt:       app.CreatedAt.Unix(),
		HasSecret:              app.HasSecret() && app.ClientSecretHash != "",
		SecretHint:             app.SecretHint,
		SecretCreatedAt:        app.SecretCreatedAt,
		TokenAuthMethod:        app.TokenAuthMethod,
		GrantTypes:             nonNil(app.GrantTypes),
		ResponseTypes:          app.ResponseTypes(),
		RedirectURIs:           nonNil(app.RedirectURIs),
		PostLogoutRedirectURIs: nonNil(app.PostLogoutRedirectURIs),
		BackchannelLogoutURI:   app.BackchannelLogoutURI,
		FrontchannelLogoutURI:  app.FrontchannelLogoutURI,
		Scopes:                 nonNil(app.Scopes),
		RequirePKCE:            app.RequirePKCE,
		RequirePushedRequests:  app.RequirePushedRequests,
		AccessTokenLifetime:    app.AccessTokenLifetime,
		IDTokenLifetime:        app.IDTokenLifetime,
		RefreshTokenLifetime:   app.RefreshTokenLifetime,
		AssertRoles:            app.AssertRoles,
		RequireRoleAssignment:  app.RequireRoleAssignment,
		IsEnabled:              app.IsEnabled,
		AllowRegistration:      app.AllowRegistration,
		LoginFlowID:            app.LoginFlowID,
		RoleCount:              roles,
		CreatedAt:              app.CreatedAt,
		UpdatedAt:              app.UpdatedAt,
	}
}

// secretResponse is an application with a just-issued secret, the only time it
// is shown.
type secretResponse struct {
	Application  applicationResponse `json:"application"`
	ClientSecret string              `json:"client_secret,omitempty"`
}

func newSecretResponse(app applicationResponse, secret string) secretResponse {
	return secretResponse{Application: app, ClientSecret: secret}
}

type pageResponse struct {
	Applications []applicationResponse `json:"applications"`
	Total        int64                 `json:"total"`
	Limit        int                   `json:"limit"`
	Offset       int                   `json:"offset"`
}

func newPageResponse(apps []model.Application, counts map[uuid.UUID]int64, total int64, query store.ApplicationQuery) pageResponse {
	out := make([]applicationResponse, 0, len(apps))
	for _, app := range apps {
		out = append(out, newApplicationResponse(app, counts[app.ID]))
	}

	return pageResponse{Applications: out, Total: total, Limit: query.Limit, Offset: query.Offset}
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// accessScope is one scope of an API, and whether the application may ask for
// it.
type accessScope struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Allowed     bool      `json:"allowed"`
}

// accessAPI is what an application may do with one API.
type accessAPI struct {
	ID           uuid.UUID     `json:"id"`
	Name         string        `json:"name"`
	Identifier   string        `json:"identifier"`
	EnforceRoles bool          `json:"enforce_roles"`
	Authorized   bool          `json:"authorized"`
	Scopes       []accessScope `json:"scopes"`
}

// accessResponse is every API, with what the application may do with each.
type accessResponse struct {
	APIs []accessAPI `json:"apis"`
}

func newAccessResponse(access []store.APIAccess) accessResponse {
	out := accessResponse{APIs: make([]accessAPI, 0, len(access))}

	for _, it := range access {
		api := accessAPI{
			ID:           it.API.ID,
			Name:         it.API.Name,
			Identifier:   it.API.Identifier,
			EnforceRoles: it.API.EnforceRoles,
			Authorized:   it.Authorized,
			Scopes:       make([]accessScope, 0, len(it.API.Scopes)),
		}

		for _, scope := range it.API.Scopes {
			api.Scopes = append(api.Scopes, accessScope{
				ID:          scope.ID,
				Name:        scope.Name,
				Description: scope.Description,
				Allowed:     slices.Contains(it.Allowed, scope.ID),
			})
		}

		out.APIs = append(out.APIs, api)
	}

	return out
}
