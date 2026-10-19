// Package admins manages administrator accounts and the panel's sign-in
// security settings.
package admins

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/kit/audit"
	"loginer/internal/api/kit/respond"
	"loginer/internal/api/kit/session"
	"loginer/internal/api/kit/validate"
	"loginer/internal/auth"
	"loginer/internal/model"
	"loginer/internal/store"
)

type Handler struct {
	store *store.Store
	auth  *auth.Service
	audit audit.Recorder
	log   *slog.Logger
}

func New(st *store.Store, service *auth.Service, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, auth: service, audit: recorder, log: log}
}

// List returns a page of administrators, newest first.
func (h *Handler) List(c *gin.Context) {
	query := listQuery(c)

	admins, total, err := h.store.Admins(c.Request.Context(), query)
	if err != nil {
		respond.Failure(c, h.log, err, "listing administrators failed")
		return
	}

	c.JSON(http.StatusOK, newPageResponse(admins, total, query))
}

func (h *Handler) Get(c *gin.Context) {
	admin, ok := h.find(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"admin": newAdminResponse(*admin)})
}

// Create adds an administrator, with the password and roles the super admin
// chose for them.
func (h *Handler) Create(c *gin.Context) {
	var req adminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	admin, err := h.build(c, &req, nil)
	if err != nil {
		respond.Failure(c, h.log, err, "checking an administrator failed")
		return
	}

	if err := h.store.CreateAdmin(c.Request.Context(), admin); err != nil {
		respond.Taken(c, h.log, err, respond.AdminEmailTaken, "creating administrator failed")
		return
	}

	h.audit.Record(c, "admin.created", targetType, admin.ID.String())

	h.answer(c, http.StatusCreated, admin)
}

// Update replaces an administrator's details, status, roles and optionally
// password. A new password, or losing the right to sign in, ends their
// sessions.
func (h *Handler) Update(c *gin.Context) {
	admin, ok := h.find(c)
	if !ok {
		return
	}

	var req adminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if _, err := h.build(c, &req, admin); err != nil {
		respond.Failure(c, h.log, err, "checking an administrator failed")
		return
	}

	ctx := c.Request.Context()

	if err := h.store.SaveAdmin(ctx, admin); err != nil {
		respond.Taken(c, h.log, err, respond.AdminEmailTaken, "updating administrator failed")
		return
	}

	h.audit.Record(c, "admin.updated", targetType, admin.ID.String())
	if req.Password != "" {
		h.audit.Record(c, "admin.password_changed", targetType, admin.ID.String())
	}

	// Your own sessions are left alone, or saving your own password here
	// would sign you out of the page you saved it from.
	changedAccess := req.Password != "" || admin.Status != model.StatusActive
	if changedAccess && admin.ID != session.Admin(c).ID {
		if err := h.store.RevokeAdminSessionsFor(ctx, admin.ID, time.Now()); err != nil {
			respond.Failure(c, h.log, err, "ending the administrator's sessions failed")
			return
		}
	}

	h.answer(c, http.StatusOK, admin)
}

func (h *Handler) answer(c *gin.Context, status int, admin *model.Admin) {
	stored, err := h.store.AdminByID(c.Request.Context(), admin.ID)
	if err != nil {
		respond.Failure(c, h.log, err, "reading the administrator back failed")
		return
	}

	c.JSON(status, gin.H{"admin": newAdminResponse(*stored)})
}

// Security returns the administrators' sign-in settings and how many have an
// authenticator.
func (h *Handler) Security(c *gin.Context) {
	h.answerSecurity(c)
}

// UpdateSecurity changes those settings. Requiring a second factor applies on
// each administrator's next page load.
func (h *Handler) UpdateSecurity(c *gin.Context) {
	var req securityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	security, err := h.store.AdminSecurity(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the admin security settings failed")
		return
	}

	was := security.RequireMFA
	security.RequireMFA = validate.Flag(req.RequireMFA, security.RequireMFA)

	if err := h.store.SaveAdminSecurity(c.Request.Context(), security); err != nil {
		respond.Failure(c, h.log, err, "saving the admin security settings failed")
		return
	}

	if was != security.RequireMFA {
		h.audit.RecordWith(c, "admin_security.updated", "admin_security", "", map[string]any{
			"require_mfa": security.RequireMFA,
		})
	}

	h.answerSecurity(c)
}

func (h *Handler) answerSecurity(c *gin.Context) {
	ctx := c.Request.Context()

	security, err := h.store.AdminSecurity(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "loading the admin security settings failed")
		return
	}

	total, withMFA, err := h.store.AdminMFACounts(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "counting administrators failed")
		return
	}

	c.JSON(http.StatusOK, newSecurityResponse(*security, total, withMFA))
}

// ResetMFA removes another administrator's second factor and signs them out
// everywhere. Your own is managed from your profile with a code.
func (h *Handler) ResetMFA(c *gin.Context) {
	admin, ok := h.find(c)
	if !ok {
		return
	}

	if admin.ID == session.Admin(c).ID {
		respond.Fail(c, ownMFA)
		return
	}

	if err := h.auth.ResetMFA(c.Request.Context(), admin); err != nil {
		respond.Failure(c, h.log, err, "resetting two-factor sign-in failed")
		return
	}

	h.audit.Record(c, "admin.mfa_reset", targetType, admin.ID.String())

	h.answer(c, http.StatusOK, admin)
}

// Delete removes an administrator for good. Nobody can remove themselves:
// that is how a panel ends up with no one able to manage it.
func (h *Handler) Delete(c *gin.Context) {
	admin, ok := h.find(c)
	if !ok {
		return
	}

	if admin.ID == session.Admin(c).ID {
		respond.Fail(c, ownAccount)
		return
	}

	if err := h.store.DeleteAdmin(c.Request.Context(), admin); err != nil {
		respond.Failure(c, h.log, err, "deleting administrator failed")
		return
	}

	h.audit.Record(c, "admin.deleted", targetType, admin.ID.String())

	c.Status(http.StatusNoContent)
}

// build checks a submitted administrator and returns the account to write. A
// super admin may not suspend themselves or drop their own panel-wide
// super_admin role.
func (h *Handler) build(c *gin.Context, req *adminRequest, into *model.Admin) (*model.Admin, error) {
	if err := req.validate(into == nil); err != nil {
		return nil, err
	}

	assignments, err := h.assignments(c, req.Assignments)
	if err != nil {
		return nil, err
	}

	status := model.Status(req.Status)

	if into != nil && into.ID == session.Admin(c).ID {
		keepsSuper := slices.ContainsFunc(assignments, func(a model.AdminRoleAssignment) bool {
			return a.Global() && a.Role.IsSuperAdmin()
		})

		if status != model.StatusActive || !keepsSuper {
			return nil, respond.Fault{
				Status:  http.StatusConflict,
				Message: "you cannot suspend yourself or take away your own super_admin role",
			}
		}
	}

	admin := into
	if admin == nil {
		admin = &model.Admin{}
	}

	if req.Password != "" {
		err := admin.SetPassword(req.Password)
		if errors.Is(err, model.ErrPasswordTooLong) {
			return nil, respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
		}
		if err != nil {
			return nil, err
		}

		// A password set by a super admin is the way back into a locked
		// account, so the wrong guesses that locked it are forgotten.
		admin.FailedLoginCount = 0
		admin.LockedUntil = nil
	}

	admin.Username = req.Email
	admin.Email = req.Email
	admin.FirstName = req.FirstName
	admin.LastName = req.LastName
	admin.Status = status
	admin.Assignments = assignments

	return admin, nil
}

func (h *Handler) assignments(c *gin.Context, requested []assignmentRequest) ([]model.AdminRoleAssignment, error) {
	ctx := c.Request.Context()

	roleIDs := make([]uuid.UUID, 0, len(requested))
	for _, a := range requested {
		roleIDs = append(roleIDs, a.RoleID)
	}

	roles, err := h.store.AdminRolesByID(ctx, roleIDs)
	if errors.Is(err, store.ErrNotFound) {
		return nil, respond.Fault{Status: http.StatusBadRequest, Message: "assignments: a role does not exist"}
	}
	if err != nil {
		return nil, err
	}

	byID := make(map[uuid.UUID]model.AdminRole, len(roles))
	for _, role := range roles {
		byID[role.ID] = role
	}

	out := []model.AdminRoleAssignment{}
	seen := map[string]bool{}

	for _, a := range requested {
		role := byID[a.RoleID]

		key := a.RoleID.String()
		if a.ApplicationID != nil {
			if role.IsSuperAdmin() {
				return nil, respond.Fault{
					Status:  http.StatusBadRequest,
					Message: "assignments: super_admin can only be held for the whole panel",
				}
			}

			if _, err := h.store.Application(ctx, *a.ApplicationID); err != nil {
				if errors.Is(err, store.ErrNotFound) {
					return nil, respond.Fault{Status: http.StatusBadRequest, Message: "assignments: an application does not exist"}
				}
				return nil, err
			}

			key += "/" + a.ApplicationID.String()
		}

		if seen[key] {
			continue
		}
		seen[key] = true

		out = append(out, model.AdminRoleAssignment{RoleID: role.ID, Role: role, ApplicationID: a.ApplicationID})
	}

	return out, nil
}

func (h *Handler) find(c *gin.Context) (*model.Admin, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, adminAbsent)
		return nil, false
	}

	admin, err := h.store.AdminByID(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, adminAbsent)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading administrator failed")
		return nil, false
	}

	return admin, true
}
