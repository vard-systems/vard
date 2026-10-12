package organization

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"loginer/internal/api/audit"
	"loginer/internal/api/respond"
	"loginer/internal/store"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger
}

// New returns a Handler.
func New(st *store.Store, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, audit: recorder, log: log}
}

// Get returns the organisation.
func (h *Handler) Get(c *gin.Context) {
	organization, err := h.store.Organization(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the organization failed")
		return
	}

	c.JSON(http.StatusOK, newResponse(*organization))
}

// Update changes the settings. What a request leaves out is left as it is,
// so a client that knows about one field does not clear the rest.
func (h *Handler) Update(c *gin.Context) {
	// Read from the database, not the cache: this is written back.
	organization, err := h.store.OrganizationForUpdate(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the organization failed")
		return
	}

	var req organizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	changed, err := req.applyTo(organization)
	if err != nil {
		respond.Failure(c, h.log, err, "validating the organization failed")
		return
	}

	if len(changed) == 0 {
		c.JSON(http.StatusOK, newResponse(*organization))
		return
	}

	if err := h.store.SaveOrganization(c.Request.Context(), organization); err != nil {
		respond.Failure(c, h.log, err, "updating the organization failed")
		return
	}

	h.audit.RecordWith(c, "organization.updated", targetType, organization.Slug, map[string]any{
		"fields": changed,
	})

	c.JSON(http.StatusOK, newResponse(*organization))
}
