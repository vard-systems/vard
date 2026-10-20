// Package organization manages the organisation's name, look and agreements
// that the sign-in pages show.
package organization

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"loginer/internal/api/kit/audit"
	"loginer/internal/api/kit/respond"
	"loginer/internal/model"
	"loginer/internal/store"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger
}

func New(st *store.Store, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, audit: recorder, log: log}
}

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
	var req organizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	var changed []string
	organization, err := h.store.UpdateOrganization(c.Request.Context(), func(organization *model.Organization) (bool, error) {
		var err error
		changed, err = req.applyTo(organization)
		return len(changed) > 0, err
	})
	if err != nil {
		respond.Failure(c, h.log, err, "updating the organization failed")
		return
	}

	if len(changed) == 0 {
		c.JSON(http.StatusOK, newResponse(*organization))
		return
	}

	h.audit.RecordWith(c, "organization.updated", targetType, organization.Slug, map[string]any{
		"fields": changed,
	})

	c.JSON(http.StatusOK, newResponse(*organization))
}
