package applications

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/audit"
	"loginer/internal/api/respond"
	"loginer/internal/api/session"
	"loginer/internal/brand"
	"loginer/internal/model"
	"loginer/internal/store"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger

	issuer string
}

// New returns a Handler. `issuer` is LOGINER_ISSUER, which tokens name the
// server by.
func New(st *store.Store, recorder audit.Recorder, log *slog.Logger, issuer string) *Handler {
	return &Handler{store: st, audit: recorder, log: log, issuer: issuer}
}

// List returns a page of the applications the administrator can see, sorted
// by name, each with how many roles it defines.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	query := listQuery(c)
	query.Only = session.Reach(c, model.PermApplicationsRead)

	apps, total, err := h.store.Applications(ctx, query)
	if err != nil {
		respond.Failure(c, h.log, err, "listing applications failed")
		return
	}

	counts, err := h.store.ApplicationRoleCounts(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "counting application roles failed")
		return
	}

	c.JSON(http.StatusOK, newPageResponse(apps, counts, total, query))
}

// Get returns one application.
func (h *Handler) Get(c *gin.Context) {
	app, ok := h.find(c, model.PermApplicationsRead)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"application": h.withCount(c, app)})
}

// Create registers an application. A confidential client is given its secret
// here, in the answer, and never again: only its hash is kept.
func (h *Handler) Create(c *gin.Context) {
	var req applicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	app := &model.Application{
		Type:              model.ApplicationType(req.Type),
		IsEnabled:         true,
		AssertRoles:       true,
		RequirePKCE:       true,
		AllowRegistration: true,
	}
	if err := req.applyTo(app, true); err != nil {
		respond.Failure(c, h.log, err, "checking an application failed")
		return
	}

	clientID, err := model.NewClientID()
	if err != nil {
		respond.Failure(c, h.log, err, "making a client id failed")
		return
	}
	app.ClientID = clientID

	secret, err := issueIfNeeded(app)
	if err != nil {
		respond.Failure(c, h.log, err, "making a client secret failed")
		return
	}

	if err := h.store.CreateApplication(c.Request.Context(), app); err != nil {
		respond.Failure(c, h.log, err, "creating application failed")
		return
	}

	h.audit.Record(c, "application.created", targetType, app.ID.String())

	c.JSON(http.StatusCreated, newSecretResponse(h.withCount(c, app), secret))
}

// Update replaces an application's settings. Its type and client id stay as
// they are: every copy of the app already in use depends on both.
func (h *Handler) Update(c *gin.Context) {
	app, ok := h.find(c, model.PermApplicationsWrite)
	if !ok {
		return
	}
	if !h.changeable(c, app) {
		return
	}

	var req applicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if req.Type != "" && model.ApplicationType(req.Type) != app.Type {
		respond.BadRequest(c, "type cannot be changed: register a new application instead")
		return
	}

	if err := req.applyTo(app, false); err != nil {
		respond.Failure(c, h.log, err, "checking an application failed")
		return
	}

	secret, err := issueIfNeeded(app)
	if err != nil {
		respond.Failure(c, h.log, err, "making a client secret failed")
		return
	}

	if err := h.store.SaveApplication(c.Request.Context(), app); err != nil {
		respond.Failure(c, h.log, err, "updating application failed")
		return
	}

	h.audit.Record(c, "application.updated", targetType, app.ID.String())

	c.JSON(http.StatusOK, newSecretResponse(h.withCount(c, app), secret))
}

// RotateSecret replaces a confidential client's secret. The old one stops
// working at once, so the app has to be given the new one straight away.
func (h *Handler) RotateSecret(c *gin.Context) {
	app, ok := h.find(c, model.PermApplicationsWrite)
	if !ok {
		return
	}
	if !h.changeable(c, app) {
		return
	}

	secret, err := app.IssueSecret(time.Now())
	if errors.Is(err, model.ErrNoSecret) {
		respond.BadRequest(c, "a public client has no secret to rotate")
		return
	}
	if err != nil {
		respond.Failure(c, h.log, err, "making a client secret failed")
		return
	}

	if err := h.store.SaveApplication(c.Request.Context(), app); err != nil {
		respond.Failure(c, h.log, err, "saving the new secret failed")
		return
	}

	h.audit.Record(c, "application.secret_rotated", targetType, app.ID.String())

	c.JSON(http.StatusOK, newSecretResponse(h.withCount(c, app), secret))
}

// Delete removes an application, the roles it defines, and everyone's hold on
// them. Tokens it has already issued stay valid until they expire.
func (h *Handler) Delete(c *gin.Context) {
	app, ok := h.find(c, model.PermApplicationsWrite)
	if !ok {
		return
	}

	// The server makes admin-cli again on its next start, so deleting it
	// would only lose its scopes; turning it off is how to stop it.
	if app.ClientID == model.AdminCLIClientID {
		respond.Fail(c, systemApplication)
		return
	}

	if err := h.store.DeleteApplication(c.Request.Context(), app); err != nil {
		respond.Failure(c, h.log, err, "deleting application failed")
		return
	}

	h.audit.Record(c, "application.deleted", targetType, app.ID.String())

	c.Status(http.StatusNoContent)
}

var systemApplication = respond.Define(http.StatusConflict, "system_application", respond.Admin)

// changeable reports whether the administrator may change this application or
// rotate its secret, answering the request when not. For an application the
// admin API trusts, its secret carries every permission a super admin granted
// it, so only a super admin may change it.
func (h *Handler) changeable(c *gin.Context, app *model.Application) bool {
	if session.Admin(c).IsSuperAdmin() {
		return true
	}

	audience, err := h.store.AudienceFor(c.Request.Context(), app.ID, brand.AdminAPIIdentifier)
	if err != nil {
		// The admin API is made at startup, so not finding it is as much a
		// fault as any other — and the safe answer to a fault is no.
		respond.Failure(c, h.log, err, "checking an application's access to the admin API failed")
		return false
	}
	if audience.Authorized {
		respond.Fail(c, systemApplicationChange)
		return false
	}

	return true
}

var systemApplicationChange = respond.Define(http.StatusForbidden, "system_application_change", respond.Admin)

// APIAccess lists every API with what the application may do with it:
// whether it is authorised to ask for tokens for it, and which scopes.
func (h *Handler) APIAccess(c *gin.Context) {
	app, ok := h.find(c, model.PermApplicationsRead)
	if !ok {
		return
	}

	access, err := h.store.ApplicationAPIAccess(c.Request.Context(), app.ID)
	if err != nil {
		respond.Failure(c, h.log, err, "listing API access failed")
		return
	}

	c.JSON(http.StatusOK, newAccessResponse(access))
}

// AuthorizeAPI lets the application request tokens for an API and replaces the
// scopes it may ask for: the ceiling on its tokens.
func (h *Handler) AuthorizeAPI(c *gin.Context) {
	ctx := c.Request.Context()

	app, ok := h.find(c, model.PermApplicationsWrite)
	if !ok {
		return
	}

	api, ok := h.findAPI(c)
	if !ok {
		return
	}

	// System API scopes are power over this server itself, so only a super
	// admin may grant them to an application.
	if api.System != "" && !session.Admin(c).IsSuperAdmin() {
		respond.Fail(c, systemAPIAccess)
		return
	}

	var req authorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	for _, id := range req.Scopes {
		if !slices.ContainsFunc(api.Scopes, func(scope model.APIScope) bool { return scope.ID == id }) {
			respond.BadRequest(c, "scopes: one of them is not a scope of "+api.Identifier)
			return
		}
	}

	if err := h.store.AuthorizeApplicationAPI(ctx, app.ID, api.ID, req.Scopes); err != nil {
		respond.Failure(c, h.log, err, "authorising the API failed")
		return
	}

	h.audit.RecordWith(c, "application.api_authorized", targetType, app.ID.String(), apiMetadata(app, api))

	h.APIAccess(c)
}

var systemAPIAccess = respond.Define(http.StatusForbidden, "system_api_access", respond.Admin)

// RevokeAPI stops the application asking for tokens for an API.
func (h *Handler) RevokeAPI(c *gin.Context) {
	app, ok := h.find(c, model.PermApplicationsWrite)
	if !ok {
		return
	}

	api, ok := h.findAPI(c)
	if !ok {
		return
	}

	// Revoking system API access is super-admin only too, so admin-cli cannot
	// be cut off by someone who could not restore it.
	if api.System != "" && !session.Admin(c).IsSuperAdmin() {
		respond.Fail(c, systemAPIAccess)
		return
	}

	if err := h.store.RevokeApplicationAPI(c.Request.Context(), app.ID, api.ID); err != nil {
		respond.Failure(c, h.log, err, "revoking the API failed")
		return
	}

	h.audit.RecordWith(c, "application.api_revoked", targetType, app.ID.String(), apiMetadata(app, api))

	h.APIAccess(c)
}

// TokenPreview runs the token endpoint's evaluation without issuing anything:
// whether a token is issued, each scope decision, and the claims.
func (h *Handler) TokenPreview(c *gin.Context) {
	ctx := c.Request.Context()

	app, ok := h.find(c, model.PermApplicationsRead)
	if !ok {
		return
	}

	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	request := model.TokenRequest{
		Application: *app,
		Requested:   strings.Fields(req.Scope),
		Issuer:      h.issuer,
		Now:         time.Now(),
	}

	if req.UserID != nil {
		if !session.Admin(c).HasPermission(model.PermUsersRead) {
			respond.Failure(c, h.log, respond.Forbidden, "")
			return
		}

		user, err := h.store.User(ctx, *req.UserID)
		if errors.Is(err, store.ErrNotFound) {
			respond.BadRequest(c, "user_id: no such user")
			return
		}
		if err != nil {
			respond.Failure(c, h.log, err, "loading the user failed")
			return
		}

		roles, err := h.store.EffectiveRoles(ctx, user)
		if err != nil {
			respond.Failure(c, h.log, err, "resolving roles failed")
			return
		}

		request.User = user
		request.Roles = roles
	}

	if identifier := strings.TrimSpace(req.Audience); identifier != "" {
		audience, err := h.store.AudienceFor(ctx, app.ID, identifier)
		if errors.Is(err, store.ErrNotFound) {
			respond.BadRequest(c, "audience: no API has that identifier")
			return
		}
		if err != nil {
			respond.Failure(c, h.log, err, "loading the audience failed")
			return
		}

		request.API = audience.API
		request.Authorized = audience.Authorized
		request.Allowed = audience.Allowed
	}

	c.JSON(http.StatusOK, gin.H{"preview": model.EvaluateToken(request)})
}

func apiMetadata(app *model.Application, api *model.API) map[string]any {
	return map[string]any{
		"api_id":      api.ID.String(),
		"api":         api.Identifier,
		"application": app.Name,
	}
}

func (h *Handler) findAPI(c *gin.Context) (*model.API, bool) {
	id, err := uuid.Parse(c.Param("api"))
	if err != nil {
		respond.BadRequest(c, "that is not an API id")
		return nil, false
	}

	api, err := h.store.API(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.NotFound(c, "no such API")
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading API failed")
		return nil, false
	}

	return api, true
}

func (h *Handler) withCount(c *gin.Context, app *model.Application) applicationResponse {
	counts, err := h.store.ApplicationRoleCounts(c.Request.Context())
	if err != nil {
		h.log.Error("counting application roles failed", "error", err)
	}

	return newApplicationResponse(*app, counts[app.ID])
}

// find loads the application in the path and checks `permission` for it. One
// the administrator cannot read is not found rather than forbidden.
func (h *Handler) find(c *gin.Context, permission string) (*model.Application, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.BadRequest(c, "that is not an application id")
		return nil, false
	}

	if !session.Allowed(c, model.PermApplicationsRead, id) {
		respond.NotFound(c, "no such application")
		return nil, false
	}

	if !session.Allowed(c, permission, id) {
		respond.Failure(c, h.log, respond.Forbidden, "")
		return nil, false
	}

	app, err := h.store.Application(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.NotFound(c, "no such application")
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading application failed")
		return nil, false
	}

	return app, true
}

func issueIfNeeded(app *model.Application) (string, error) {
	if !app.HasSecret() || app.ClientSecretHash != "" {
		return "", nil
	}

	return app.IssueSecret(time.Now())
}
