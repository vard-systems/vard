// Package apis manages the APIs applications request tokens for: their scopes,
// the applications allowed to call them, and their logs.
package apis

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/internal/application/oidc"
	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
	"vard/internal/interfaces/api/kit/audit"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/session"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger

	issuer string
}

// New returns a Handler. `issuer` is VARD_ISSUER, which tokens name the
// server by.
func New(st *store.Store, recorder audit.Recorder, log *slog.Logger, issuer string) *Handler {
	return &Handler{store: st, audit: recorder, log: log, issuer: issuer}
}

// List returns every API matching the search, sorted by name, with its scopes
// and how many applications may use it.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	apis, err := h.store.APIs(ctx, c.Query("search"))
	if err != nil {
		respond.Failure(c, h.log, err, "listing APIs failed")
		return
	}

	details, err := h.details(c)
	if err != nil {
		respond.Failure(c, h.log, err, "counting API applications and roles failed")
		return
	}

	c.JSON(http.StatusOK, newListResponse(apis, details))
}

func (h *Handler) Get(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"api": h.withCount(c, api)})
}

// Create registers an API with its scopes.
func (h *Handler) Create(c *gin.Context) {
	var req apiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	api := &model.API{EnforceRoles: true}
	if err := req.applyTo(api, true); err != nil {
		respond.Failure(c, h.log, err, "checking an API failed")
		return
	}

	if err := h.store.CreateAPI(c.Request.Context(), api); err != nil {
		respond.Taken(c, h.log, err, identifierTaken, "creating API failed")
		return
	}

	h.audit.Record(c, "api.created", targetType, api.ID.String())

	h.answer(c, http.StatusCreated, api.ID)
}

// Update replaces an API's name, description, role enforcement and scopes. The
// identifier never changes.
func (h *Handler) Update(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	var req apiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if req.Identifier != "" && req.Identifier != api.Identifier {
		respond.Fail(c, identifierFixed)
		return
	}

	// A system API's scopes stay the server's whatever the request says; only
	// its name, description and token settings change.
	scopes := api.Scopes

	if err := req.applyTo(api, false); err != nil {
		respond.Failure(c, h.log, err, "checking an API failed")
		return
	}

	if api.System != "" {
		api.Scopes = scopes
	}

	if err := h.store.SaveAPI(c.Request.Context(), api); err != nil {
		respond.Taken(c, h.log, err, identifierTaken, "updating API failed")
		return
	}

	h.audit.Record(c, "api.updated", targetType, api.ID.String())

	h.answer(c, http.StatusOK, api.ID)
}

var systemAPI = respond.Define(http.StatusConflict, "system_api", respond.Admin)

// Delete removes an API with its scopes, application authorisations and role
// grants. Issued tokens stay valid until they expire.
func (h *Handler) Delete(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	if api.System != "" {
		respond.Fail(c, systemAPI)
		return
	}

	if err := h.store.DeleteAPI(c.Request.Context(), api); err != nil {
		respond.Failure(c, h.log, err, "deleting API failed")
		return
	}

	h.audit.Record(c, "api.deleted", targetType, api.ID.String())

	c.Status(http.StatusNoContent)
}

func (h *Handler) answer(c *gin.Context, status int, id uuid.UUID) {
	api, err := h.store.API(c.Request.Context(), id)
	if err != nil {
		respond.Failure(c, h.log, err, "reading the API back failed")
		return
	}

	c.JSON(status, gin.H{"api": h.withCount(c, api)})
}

// Applications lists visible applications with their access to the API. Access
// is changed through the application's endpoints.
func (h *Handler) Applications(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	apps, err := h.store.APIApplications(c.Request.Context(), api.ID, session.Reach(c, model.PermApplicationsRead))
	if err != nil {
		respond.Failure(c, h.log, err, "listing the API's applications failed")
		return
	}

	c.JSON(http.StatusOK, newApplicationsResponse(apps))
}

// Introspectors lists the visible applications, each saying whether it may
// introspect the API's access tokens whichever client they were issued to.
func (h *Handler) Introspectors(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	apps, err := h.store.APIIntrospectors(c.Request.Context(), api.ID, session.Reach(c, model.PermApplicationsRead))
	if err != nil {
		respond.Failure(c, h.log, err, "listing the API's introspectors failed")
		return
	}

	c.JSON(http.StatusOK, newIntrospectorsResponse(apps))
}

// SetIntrospectors replaces the applications that may introspect the API's
// access tokens. Each has to be confidential; an empty list leaves the API's
// tokens answered only to the clients they were issued to.
func (h *Handler) SetIntrospectors(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	var req introspectorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	ctx := c.Request.Context()
	found, _, err := h.store.Applications(ctx, store.ApplicationQuery{Only: req.ApplicationIDs, Limit: maxIntrospectors + 1})
	if err != nil {
		respond.Failure(c, h.log, err, "loading the applications failed")
		return
	}
	ids, err := req.introspectors(found)
	if err != nil {
		respond.Failure(c, h.log, err, "checking the API's introspectors failed")
		return
	}

	if err := h.store.SetAPIIntrospectors(ctx, api.ID, ids, time.Now()); err != nil {
		respond.Failure(c, h.log, err, "saving the API's introspectors failed")
		return
	}

	h.audit.RecordWith(c, "api.introspectors_updated", targetType, api.ID.String(), map[string]any{
		"api": api.Name, "introspectors": len(ids),
	})

	apps, err := h.store.APIIntrospectors(ctx, api.ID, session.Reach(c, model.PermApplicationsRead))
	if err != nil {
		respond.Failure(c, h.log, err, "listing the API's introspectors failed")
		return
	}
	c.JSON(http.StatusOK, newIntrospectorsResponse(apps))
}

// Logs lists changes to the API and applications gaining or losing access,
// newest first.
func (h *Handler) Logs(c *gin.Context) {
	api, ok := h.find(c)
	if !ok {
		return
	}

	events, err := h.store.APIAuditLog(c.Request.Context(), api.ID, maxLogEntries)
	if err != nil {
		respond.Failure(c, h.log, err, "reading the API's log failed")
		return
	}

	c.JSON(http.StatusOK, newLogResponse(events))
}

func (h *Handler) details(c *gin.Context) (apiDetails, error) {
	ctx := c.Request.Context()

	applications, err := h.store.APIApplicationCounts(ctx)
	if err != nil {
		return apiDetails{}, err
	}

	roles, err := h.store.APIRoleCounts(ctx)
	if err != nil {
		return apiDetails{}, err
	}

	return apiDetails{
		applications: applications,
		roles:        roles,
		issuer:       h.issuer,
		jwksURI:      h.issuer + oidc.PathJWKS,
	}, nil
}

func (h *Handler) withCount(c *gin.Context, api *model.API) apiResponse {
	details, err := h.details(c)
	if err != nil {
		h.log.Error("counting API applications and roles failed", "error", err)
		details = apiDetails{issuer: h.issuer, jwksURI: h.issuer + oidc.PathJWKS}
	}

	return newAPIResponse(*api, details)
}

func (h *Handler) find(c *gin.Context) (*model.API, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, respond.APINotFound)
		return nil, false
	}

	api, err := h.store.API(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, respond.APINotFound)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading API failed")
		return nil, false
	}

	return api, true
}
