package adminroles

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/audit"
	"loginer/internal/api/respond"
	"loginer/internal/model"
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

// Permissions returns the catalog roles pick from, in the order the panel
// lists it.
func (h *Handler) Permissions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"permissions": model.AdminPermissions})
}

// List returns every admin role matching the search, sorted by name, with how
// many administrators hold each.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	roles, err := h.store.AdminRoles(ctx, c.Query("search"))
	if err != nil {
		respond.Failure(c, h.log, err, "listing admin roles failed")
		return
	}

	counts, err := h.store.AdminRoleMemberCounts(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "counting admin role members failed")
		return
	}

	c.JSON(http.StatusOK, newListResponse(roles, counts))
}

// Create adds an admin role.
func (h *Handler) Create(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	role := &model.AdminRole{}
	if err := req.applyTo(role); err != nil {
		respond.Failure(c, h.log, err, "checking an admin role failed")
		return
	}

	if err := h.store.CreateAdminRole(c.Request.Context(), role); err != nil {
		h.respondWrite(c, err, "creating admin role failed")
		return
	}

	h.audit.Record(c, "admin_role.created", targetType, role.ID.String())

	c.JSON(http.StatusCreated, gin.H{"role": h.withCount(c, role)})
}

// Update replaces an admin role's name, description and permissions. Every
// administrator holding it has the new permissions on their next request.
func (h *Handler) Update(c *gin.Context) {
	role, ok := h.findEditable(c)
	if !ok {
		return
	}

	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if err := req.applyTo(role); err != nil {
		respond.Failure(c, h.log, err, "checking an admin role failed")
		return
	}

	if err := h.store.SaveAdminRole(c.Request.Context(), role); err != nil {
		h.respondWrite(c, err, "updating admin role failed")
		return
	}

	h.audit.Record(c, "admin_role.updated", targetType, role.ID.String())

	c.JSON(http.StatusOK, gin.H{"role": h.withCount(c, role)})
}

// Delete removes an admin role. The administrators who held it keep their
// other roles.
func (h *Handler) Delete(c *gin.Context) {
	role, ok := h.findEditable(c)
	if !ok {
		return
	}

	if err := h.store.DeleteAdminRole(c.Request.Context(), role); err != nil {
		respond.Failure(c, h.log, err, "deleting admin role failed")
		return
	}

	h.audit.Record(c, "admin_role.deleted", targetType, role.ID.String())

	c.Status(http.StatusNoContent)
}

func (h *Handler) withCount(c *gin.Context, role *model.AdminRole) roleResponse {
	counts, err := h.store.AdminRoleMemberCounts(c.Request.Context())
	if err != nil {
		h.log.Error("counting admin role members failed", "error", err)
	}

	return newRoleResponse(*role, counts[role.ID])
}

func (h *Handler) findEditable(c *gin.Context) (*model.AdminRole, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, roleAbsent)
		return nil, false
	}

	role, err := h.store.AdminRole(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, roleAbsent)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading admin role failed")
		return nil, false
	case role.IsSuperAdmin():
		respond.Fail(c, superAdminFixed)
		return nil, false
	}

	return role, true
}

func (h *Handler) respondWrite(c *gin.Context, err error, note string) {
	if errors.Is(err, store.ErrDuplicate) {
		respond.Fail(c, roleNameTaken)
		return
	}

	respond.Failure(c, h.log, err, note)
}
