// Package organization manages the organisation's name, look and agreements
// that the sign-in pages show.
package organization

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/api/kit/audit"
	"vard/api/kit/respond"
	"vard/core/model"
	"vard/core/store"
)

var (
	logoTooLarge  = respond.Define(http.StatusRequestEntityTooLarge, "organization_logo_too_large", respond.Admin)
	logoType      = respond.Define(http.StatusUnsupportedMediaType, "organization_logo_type", respond.Admin)
	logoDimension = respond.Define(http.StatusBadRequest, "organization_logo_dimensions", respond.Admin)
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

// Logo sends the app logo to the panel, for any administrator: the header
// shows it on every page.
func (h *Handler) Logo(c *gin.Context) {
	logo, err := h.store.OrganizationLogo(c.Request.Context())
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, respond.OrganizationLogoNotFound)
		return
	}
	if err != nil {
		respond.Failure(c, h.log, err, "loading the organization logo failed")
		return
	}

	respond.Image(c, logo.ContentType, logo.Data, logo.Version, true)
}

// SetLogo uploads the app logo in place of the one there.
func (h *Handler) SetLogo(c *gin.Context) {
	var req logoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// A file far over the limit is cut off by the API's body limit before
		// it can be decoded; it is still a logo too large, not a broken request.
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			respond.Fail(c, logoTooLarge)
			return
		}
		respond.Fail(c, respond.InvalidBody)
		return
	}

	logo, err := req.logo()
	if err != nil {
		respond.Failure(c, h.log, err, "reading an organization logo failed")
		return
	}

	organization, err := h.store.SetOrganizationLogo(c.Request.Context(), logo)
	if err != nil {
		respond.Failure(c, h.log, err, "saving the organization logo failed")
		return
	}

	h.audit.RecordWith(c, "organization.updated", targetType, organization.Slug, map[string]any{
		"fields": []string{"logo"},
	})

	c.JSON(http.StatusOK, newResponse(*organization))
}

// RemoveLogo goes back to the panel's own logo.
func (h *Handler) RemoveLogo(c *gin.Context) {
	organization, removed, err := h.store.RemoveOrganizationLogo(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "removing the organization logo failed")
		return
	}

	if removed {
		h.audit.RecordWith(c, "organization.updated", targetType, organization.Slug, map[string]any{
			"fields": []string{"logo"},
		})
	}

	c.JSON(http.StatusOK, newResponse(*organization))
}
