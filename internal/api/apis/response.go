package apis

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/model"
	"loginer/internal/store"
)

type scopeResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
}

// apiResponse is an API as the panel sees it.
type apiResponse struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Identifier   string          `json:"identifier"`
	Description  string          `json:"description"`
	EnforceRoles bool            `json:"enforce_roles"`
	Scopes       []scopeResponse `json:"scopes"`

	// System is "admin" or "account" for an API this server is itself,
	// whose identifier and scopes the panel shows but does not change.
	System string `json:"system"`

	SigningAlgorithm   string `json:"signing_algorithm"`
	TokenLifetime      int    `json:"token_lifetime"`
	AllowOfflineAccess bool   `json:"allow_offline_access"`

	// ApplicationCount is how many applications may ask for tokens for it,
	// and RoleCount how many roles grant at least one of its scopes.
	ApplicationCount int64 `json:"application_count"`
	RoleCount        int64 `json:"role_count"`

	// Issuer and JWKSURI are what an API validating its tokens needs from
	// this server: the iss to expect, and where the signing keys are.
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// apiDetails is what a response needs beyond the API itself.
type apiDetails struct {
	applications map[uuid.UUID]int64
	roles        map[uuid.UUID]int64
	issuer       string
	jwksURI      string
}

func newAPIResponse(api model.API, details apiDetails) apiResponse {
	scopes := make([]scopeResponse, 0, len(api.Scopes))
	for _, scope := range api.Scopes {
		scopes = append(scopes, scopeResponse{
			ID:          scope.ID,
			Name:        scope.Name,
			Description: scope.Description,
			IsDefault:   scope.IsDefault,
		})
	}

	return apiResponse{
		ID:                 api.ID,
		Name:               api.Name,
		Identifier:         api.Identifier,
		System:             api.System,
		Description:        api.Description,
		EnforceRoles:       api.EnforceRoles,
		Scopes:             scopes,
		SigningAlgorithm:   api.SigningAlgorithm,
		TokenLifetime:      api.TokenLifetime,
		AllowOfflineAccess: api.AllowOfflineAccess,
		ApplicationCount:   details.applications[api.ID],
		RoleCount:          details.roles[api.ID],
		Issuer:             details.issuer,
		JWKSURI:            details.jwksURI,
		CreatedAt:          api.CreatedAt,
		UpdatedAt:          api.UpdatedAt,
	}
}

// listResponse is every API matching a search.
type listResponse struct {
	APIs  []apiResponse `json:"apis"`
	Total int           `json:"total"`
}

func newListResponse(apis []model.API, details apiDetails) listResponse {
	out := make([]apiResponse, 0, len(apis))
	for _, api := range apis {
		out = append(out, newAPIResponse(api, details))
	}

	return listResponse{APIs: out, Total: len(out)}
}

// applicationAccess is one application, and what it may do with the API.
type applicationAccess struct {
	ID         uuid.UUID             `json:"id"`
	Name       string                `json:"name"`
	Type       model.ApplicationType `json:"type"`
	ClientID   string                `json:"client_id"`
	IsEnabled  bool                  `json:"is_enabled"`
	Authorized bool                  `json:"authorized"`
	// Allowed are the ids of the API's scopes the application may ask for.
	Allowed []uuid.UUID `json:"allowed"`
}

func newApplicationsResponse(apps []store.APIApplication) gin.H {
	out := make([]applicationAccess, 0, len(apps))
	for _, it := range apps {
		out = append(out, applicationAccess{
			ID:         it.Application.ID,
			Name:       it.Application.Name,
			Type:       it.Application.Type,
			ClientID:   it.Application.ClientID,
			IsEnabled:  it.Application.IsEnabled,
			Authorized: it.Authorized,
			Allowed:    it.Allowed,
		})
	}

	return gin.H{"applications": out}
}

// logEntry is one thing that happened to or involving the API.
type logEntry struct {
	ID        uuid.UUID      `json:"id"`
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	IP        string         `json:"ip"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

func newLogResponse(events []model.AuditLog) gin.H {
	out := make([]logEntry, 0, len(events))
	for _, event := range events {
		out = append(out, logEntry{
			ID:        event.ID,
			Action:    event.Action,
			Actor:     event.ActorEmail,
			IP:        event.IP,
			Metadata:  event.Metadata,
			CreatedAt: event.CreatedAt,
		})
	}

	return gin.H{"logs": out}
}
