// Package roles manages the roles users are given, globally or per application.
package roles

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/kit/audit"
	"loginer/internal/api/kit/respond"
	"loginer/internal/api/kit/session"
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

const (
	globalScope      = "global"
	applicationScope = "application"
)

// List returns a page of roles sorted by name, each with its member count and
// every role it includes. `scope` is "global", "application" or empty for both;
// `application` narrows to one application.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()
	query := listQuery(c)

	reach := session.Reach(c, model.PermApplicationsRead)

	switch scope := c.Query("scope"); {
	case c.Query("application") != "":
		app, err := uuid.Parse(c.Query("application"))
		if err != nil {
			respond.Fail(c, respond.ApplicationNotFound)
			return
		}

		if !session.Allowed(c, model.PermApplicationsRead, app) {
			respond.Fail(c, respond.ApplicationNotFound)
			return
		}

		query.Applications = []uuid.UUID{app}
	case scope == globalScope:
		query.Global = true
		query.Applications = []uuid.UUID{}
	case scope == applicationScope:
		query.Applications = reach
	case scope == "":
		query.Global = true
		query.Applications = reach
	default:
		respond.Fail(c, scopeInvalid)
		return
	}

	roles, total, err := h.store.UserRoles(ctx, query)
	if err != nil {
		respond.Failure(c, h.log, err, "listing roles failed")
		return
	}

	details, err := h.details(c, roles)
	if err != nil {
		respond.Failure(c, h.log, err, "resolving roles failed")
		return
	}

	c.JSON(http.StatusOK, newPageResponse(roles, details, total, query))
}

func (h *Handler) Get(c *gin.Context) {
	role, ok := h.find(c, "")
	if !ok {
		return
	}

	h.answer(c, http.StatusOK, role)
}

// Create adds a role: a global one when application_id is null, otherwise
// one of that application's.
func (h *Handler) Create(c *gin.Context) {
	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if app := req.ApplicationID; app != nil {
		if !session.Allowed(c, model.PermApplicationsRead, *app) {
			respond.Fail(c, applicationUnknown)
			return
		}

		if _, err := h.store.Application(c.Request.Context(), *app); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				respond.Fail(c, applicationUnknown)
				return
			}

			respond.Failure(c, h.log, err, "loading the role's application failed")
			return
		}
	}

	if !session.AllowedScope(c, model.PermUserRolesWrite, req.ApplicationID) {
		respond.Failure(c, h.log, respond.Forbidden, "")
		return
	}

	role, err := h.build(c, &req, nil)
	if err != nil {
		respond.Failure(c, h.log, err, "checking a role failed")
		return
	}

	if err := h.store.CreateUserRole(c.Request.Context(), role); err != nil {
		respond.Taken(c, h.log, err, roleNameTaken, "creating role failed")
		return
	}

	h.audit.Record(c, "user_role.created", targetType, role.ID.String())

	h.answer(c, http.StatusCreated, role)
}

// Update replaces a role's name, description, default flag and inheritance. Its
// scope never changes.
func (h *Handler) Update(c *gin.Context) {
	role, ok := h.find(c, model.PermUserRolesWrite)
	if !ok {
		return
	}

	var req roleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if _, err := h.build(c, &req, role); err != nil {
		respond.Failure(c, h.log, err, "checking a role failed")
		return
	}

	if err := h.store.SaveUserRole(c.Request.Context(), role); err != nil {
		respond.Taken(c, h.log, err, roleNameTaken, "updating role failed")
		return
	}

	h.audit.Record(c, "user_role.updated", targetType, role.ID.String())

	h.answer(c, http.StatusOK, role)
}

// Delete removes a role for good. The users who held it keep everything else
// they hold, and the roles that included it simply stop.
func (h *Handler) Delete(c *gin.Context) {
	role, ok := h.find(c, model.PermUserRolesWrite)
	if !ok {
		return
	}

	if err := h.store.DeleteUserRole(c.Request.Context(), role); err != nil {
		respond.Failure(c, h.log, err, "deleting role failed")
		return
	}

	h.audit.Record(c, "user_role.deleted", targetType, role.ID.String())

	c.Status(http.StatusNoContent)
}

func (h *Handler) build(c *gin.Context, req *roleRequest, into *model.UserRole) (*model.UserRole, error) {
	ctx := c.Request.Context()

	if err := req.validate(); err != nil {
		return nil, err
	}

	inherits, err := h.store.UserRolesByID(ctx, req.Inherits)
	if errors.Is(err, store.ErrNotFound) {
		return nil, respond.Fault{Status: http.StatusBadRequest, Message: "inherits: one of them does not exist"}
	}
	if err != nil {
		return nil, err
	}

	role := into
	if role == nil {
		role = &model.UserRole{ApplicationID: req.ApplicationID}
	}

	for _, inherited := range inherits {
		if !session.SeesRole(c, inherited) {
			return nil, respond.Fault{Status: http.StatusBadRequest, Message: "inherits: one of them does not exist"}
		}

		// Including a role hands it to everyone holding this one, so adding it
		// needs the right to manage roles where it belongs. Already included
		// roles stay.
		added := !slices.ContainsFunc(role.Inherits, func(held model.UserRole) bool { return held.ID == inherited.ID })
		if added && !session.AllowedScope(c, model.PermUserRolesWrite, inherited.ApplicationID) {
			return nil, respond.Fault{
				Status:  http.StatusForbidden,
				Message: "inherits: your roles do not allow you to include " + inherited.Name,
			}
		}

		if !role.MayInherit(inherited) {
			return nil, respond.Fault{
				Status: http.StatusBadRequest,
				Message: "inherits: " + inherited.Name + " belongs to another application; " +
					"an application role can include global roles and its own application's roles",
			}
		}
	}

	if into != nil && len(inherits) > 0 {
		graph, err := h.store.RoleGraph(ctx)
		if err != nil {
			return nil, err
		}

		if graph.WouldCycle(role.ID, req.Inherits) {
			return nil, respond.Fault{
				Status:  http.StatusBadRequest,
				Message: "inherits: a role cannot include itself, or a role that includes it",
			}
		}
	}

	if req.APIScopes != nil {
		if !session.Admin(c).HasPermission(model.PermAPIsRead) {
			return nil, respond.Fault{
				Status:  http.StatusForbidden,
				Message: "api_scopes: your roles do not allow you to see APIs",
			}
		}

		scopes, err := h.store.APIScopesByID(ctx, *req.APIScopes)
		if errors.Is(err, store.ErrNotFound) {
			return nil, respond.Fault{Status: http.StatusBadRequest, Message: "api_scopes: one of them does not exist"}
		}
		if err != nil {
			return nil, err
		}

		role.APIScopes = scopes
	}

	role.Name = req.Name
	role.Description = req.Description
	role.IsDefault = req.IsDefault
	role.Inherits = inherits

	return role, nil
}

func (h *Handler) details(c *gin.Context, roles []model.UserRole) (roleDetails, error) {
	ctx := c.Request.Context()

	ids := make([]uuid.UUID, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.ID)
	}

	members, err := h.store.RoleMemberCounts(ctx, ids)
	if err != nil {
		return roleDetails{}, err
	}

	graph, err := h.store.RoleGraph(ctx)
	if err != nil {
		return roleDetails{}, err
	}

	return roleDetails{members: members, graph: graph, visible: func(role model.UserRole) bool {
		return session.SeesRole(c, role)
	}}, nil
}

func (h *Handler) answer(c *gin.Context, status int, role *model.UserRole) {
	details, err := h.details(c, []model.UserRole{*role})
	if err != nil {
		respond.Failure(c, h.log, err, "resolving role failed")
		return
	}

	details.graph[role.ID] = *role

	c.JSON(status, gin.H{"role": newRoleResponse(*role, details)})
}

// find loads the role in the path and, given `permission`, checks it for the
// role's scope. Invisible roles are not found.
func (h *Handler) find(c *gin.Context, permission string) (*model.UserRole, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, respond.UserRoleNotFound)
		return nil, false
	}

	role, err := h.store.UserRole(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound), err == nil && !session.SeesRole(c, *role):
		respond.Fail(c, respond.UserRoleNotFound)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading role failed")
		return nil, false
	case permission != "" && !session.AllowedScope(c, permission, role.ApplicationID):
		respond.Failure(c, h.log, respond.Forbidden, "")
		return nil, false
	}

	return role, true
}
