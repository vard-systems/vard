// Package users manages user accounts, their roles and their linked social
// accounts.
package users

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/internal/api/kit/audit"
	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/session"
	"vard/internal/api/kit/validate"
	"vard/internal/model"
	"vard/internal/store"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger
}

func New(st *store.Store, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, audit: recorder, log: log}
}

// List returns a page of users, newest first. `search` matches the email and
// any field text.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()
	query := listQuery(c)

	// Filtering by a role says who holds it, which is the role's scope's
	// business: a role the administrator cannot see is not found.
	if query.Role != nil {
		role, err := h.store.UserRole(ctx, *query.Role)
		if errors.Is(err, store.ErrNotFound) || err == nil && !session.SeesRole(c, *role) {
			respond.Fail(c, respond.UserRoleNotFound)
			return
		}
		if err != nil {
			respond.Failure(c, h.log, err, "loading the role to filter by failed")
			return
		}
	}

	users, total, err := h.store.Users(ctx, query)
	if err != nil {
		respond.Failure(c, h.log, err, "listing users failed")
		return
	}

	for i := range users {
		visibleRoles(c, &users[i])
	}

	if err := h.fillSocialAccounts(c, pointersTo(users)...); err != nil {
		respond.Failure(c, h.log, err, "listing the providers users sign in with failed")
		return
	}

	if err := h.fillTwoStep(c, pointersTo(users)...); err != nil {
		respond.Failure(c, h.log, err, "reading users' two-step verification failed")
		return
	}

	c.JSON(http.StatusOK, newPageResponse(users, total, query))
}

func (h *Handler) Get(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	visibleRoles(c, user)

	if err := h.fillSocialAccounts(c, user); err != nil {
		respond.Failure(c, h.log, err, "listing the providers this user signs in with failed")
		return
	}

	if err := h.fillTwoStep(c, user); err != nil {
		respond.Failure(c, h.log, err, "reading a user's two-step verification failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// ResetTwoStep turns a user's two-step verification off, for someone who
// lost every factor and recovery code. They sign in with their first factor
// alone until they turn it on again.
func (h *Handler) ResetTwoStep(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	removed, err := h.store.ResetTwoStep(c.Request.Context(), user.ID)
	if err != nil {
		respond.Failure(c, h.log, err, "resetting two-step verification failed")
		return
	}

	h.audit.RecordWith(c, "user.two_step_reset", targetType, user.ID.String(), map[string]any{
		"email": user.Email, "factors": removed,
	})

	c.Status(http.StatusNoContent)
}

func (h *Handler) fillSocialAccounts(c *gin.Context, users ...*model.User) error {
	ids := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}

	accounts, err := h.store.SocialAccountsFor(c.Request.Context(), ids)
	if err != nil {
		return err
	}

	for _, user := range users {
		user.SocialAccounts = accounts[user.ID]
	}

	return nil
}

func (h *Handler) fillTwoStep(c *gin.Context, users ...*model.User) error {
	ids := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.ID)
	}

	with, err := h.store.UsersWithTwoStep(c.Request.Context(), ids)
	if err != nil {
		return err
	}

	for _, user := range users {
		user.TwoStepEnabled = with[user.ID]
	}

	return nil
}

func pointersTo(users []model.User) []*model.User {
	out := make([]*model.User, len(users))
	for i := range users {
		out[i] = &users[i]
	}

	return out
}

// Disconnect removes a provider a user signs in with; their account and other
// ways in remain.
func (h *Handler) Disconnect(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	id, err := uuid.Parse(c.Param("identity"))
	if err != nil {
		respond.Fail(c, identityAbsent)
		return
	}

	identity, err := h.store.SocialIdentityByID(c.Request.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound) || err == nil && identity.UserID != user.ID:
		respond.Fail(c, identityAbsent)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "loading the connection failed")
		return
	}

	if err := h.store.DeleteSocialIdentity(c.Request.Context(), identity); err != nil {
		respond.Failure(c, h.log, err, "disconnecting a provider failed")
		return
	}

	h.audit.RecordWith(c, "user.identity_disconnected", targetType, user.ID.String(), map[string]any{
		"email": user.Email,
	})

	c.Status(http.StatusNoContent)
}

// Create adds a user.
func (h *Handler) Create(c *gin.Context) {
	var req userRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	user, err := h.build(c, &req, nil)
	if err != nil {
		respond.Failure(c, h.log, err, "checking a user failed")
		return
	}

	if err := h.store.CreateUser(c.Request.Context(), user); err != nil {
		respond.Taken(c, h.log, err, emailTaken, "creating user failed")
		return
	}

	h.audit.Record(c, "user.created", targetType, user.ID.String())

	visibleRoles(c, user)

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// Update replaces a user's email, verified flag and fields, and the password
// too when a new one is given.
func (h *Handler) Update(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	var req userRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if _, err := h.build(c, &req, user); err != nil {
		respond.Failure(c, h.log, err, "checking a user failed")
		return
	}

	if err := h.store.SaveUser(c.Request.Context(), user); err != nil {
		respond.Taken(c, h.log, err, emailTaken, "updating user failed")
		return
	}

	h.audit.Record(c, "user.updated", targetType, user.ID.String())
	if req.Password != "" {
		// A new password usually answers a takeover, so existing sessions and
		// refresh tokens end.
		sessions, tokens, err := h.store.SignOutUser(c.Request.Context(), user.ID, time.Now())
		if err != nil {
			respond.Failure(c, h.log, err, "ending the user's sessions failed")
			return
		}

		h.audit.RecordWith(c, "user.password_changed", targetType, user.ID.String(), map[string]any{
			"sessions": sessions, "tokens": tokens,
		})
	}

	visibleRoles(c, user)

	if err := h.fillSocialAccounts(c, user); err != nil {
		respond.Failure(c, h.log, err, "listing the providers this user signs in with failed")
		return
	}

	if err := h.fillTwoStep(c, user); err != nil {
		respond.Failure(c, h.log, err, "reading a user's two-step verification failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// Roles returns the roles a user holds as a token would carry them: global
// roles and each application's, direct and inherited. `client_id` narrows to
// one application; applications the administrator cannot see are left out.
func (h *Handler) Roles(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	apps, graph, ok := h.roleContext(c)
	if !ok {
		return
	}

	if clientID := c.Query("client_id"); clientID != "" {
		apps = slices.DeleteFunc(apps, func(app model.Application) bool { return app.ClientID != clientID })

		if len(apps) == 0 {
			respond.Fail(c, respond.ApplicationNotFound)
			return
		}
	}

	c.JSON(http.StatusOK, newRolesResponse(user, apps, graph))
}

// RoleMappings returns every role a user holds, direct and inherited, with what
// each comes through. Roles of invisible applications are left out.
func (h *Handler) RoleMappings(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	apps, graph, ok := h.roleContext(c)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, newMappingsResponse(user, apps, graph, func(role model.UserRole) bool {
		return session.SeesRole(c, role)
	}))
}

// AssignRoles gives a user roles directly, keeping existing ones. Each needs
// role_assignments.write for its scope; one refusal refuses the request.
func (h *Handler) AssignRoles(c *gin.Context) {
	ctx := c.Request.Context()

	var req assignRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Roles) == 0 {
		respond.Fail(c, noRoles)
		return
	}

	user, ok := h.find(c)
	if !ok {
		return
	}

	roles, err := h.store.UserRolesByID(ctx, req.Roles)
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, roleUnknown)
		return
	}
	if err != nil {
		respond.Failure(c, h.log, err, "loading roles failed")
		return
	}

	for _, role := range roles {
		if !session.SeesRole(c, role) {
			respond.Fail(c, roleUnknown)
			return
		}

		if !session.AllowedScope(c, model.PermRoleAssignmentsWrite, role.ApplicationID) {
			respond.Failure(c, h.log, respond.Forbidden, "")
			return
		}
	}

	if err := h.store.AddUserRoles(ctx, user.ID, roles); err != nil {
		respond.Failure(c, h.log, err, "assigning roles failed")
		return
	}

	h.audit.Record(c, "user.roles_assigned", targetType, user.ID.String())

	h.RoleMappings(c)
}

// UnassignRole removes a directly held role; an inherited one is removed by
// removing what it comes through.
func (h *Handler) UnassignRole(c *gin.Context) {
	ctx := c.Request.Context()

	roleID, err := uuid.Parse(c.Param("role"))
	if err != nil {
		respond.Fail(c, respond.UserRoleNotFound)
		return
	}

	user, ok := h.find(c)
	if !ok {
		return
	}

	role, err := h.store.UserRole(ctx, roleID)
	switch {
	case errors.Is(err, store.ErrNotFound), err == nil && !session.SeesRole(c, *role):
		respond.Fail(c, respond.UserRoleNotFound)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "loading role failed")
		return
	case !session.AllowedScope(c, model.PermRoleAssignmentsWrite, role.ApplicationID):
		respond.Failure(c, h.log, respond.Forbidden, "")
		return
	}

	if !slices.ContainsFunc(user.Roles, func(held model.UserRole) bool { return held.ID == role.ID }) {
		respond.Fail(c, roleNotHeld)
		return
	}

	if err := h.store.RemoveUserRole(ctx, user.ID, role.ID); err != nil {
		respond.Failure(c, h.log, err, "unassigning role failed")
		return
	}

	h.audit.Record(c, "user.role_unassigned", targetType, user.ID.String())

	h.RoleMappings(c)
}

func (h *Handler) roleContext(c *gin.Context) ([]model.Application, model.RoleGraph, bool) {
	ctx := c.Request.Context()

	apps, _, err := h.store.Applications(ctx, store.ApplicationQuery{
		Only:  session.Reach(c, model.PermApplicationsRead),
		Limit: maxApplications,
	})
	if err != nil {
		respond.Failure(c, h.log, err, "listing applications failed")
		return nil, nil, false
	}

	graph, err := h.store.RoleGraph(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "resolving roles failed")
		return nil, nil, false
	}

	return apps, graph, true
}

// Delete removes a user for good.
func (h *Handler) Delete(c *gin.Context) {
	user, ok := h.find(c)
	if !ok {
		return
	}

	if err := h.store.DeleteUser(c.Request.Context(), user); err != nil {
		respond.Failure(c, h.log, err, "deleting user failed")
		return
	}

	h.audit.Record(c, "user.deleted", targetType, user.ID.String())

	c.Status(http.StatusNoContent)
}

func (h *Handler) build(c *gin.Context, req *userRequest, into *model.User) (*model.User, error) {
	ctx := c.Request.Context()

	if err := req.validate(into == nil); err != nil {
		return nil, err
	}

	fields, err := h.store.UserFields(ctx)
	if err != nil {
		return nil, err
	}

	self := uuid.Nil
	if into != nil {
		self = into.ID
	}

	data, err := normalize(ctx, h.store, fields, req.Data, self)
	if err != nil {
		return nil, err
	}

	user := into
	if user == nil {
		user = &model.User{IsActive: true}
	}

	// A changed address is unverified whatever the form says, or a later social
	// or SSO link would trust an address nobody proved.
	changedEmail := into != nil && req.Email != user.Email

	user.Email = req.Email
	user.IsEmailVerified = validate.Flag(req.IsEmailVerified, user.IsEmailVerified)
	if changedEmail {
		user.IsEmailVerified = false
	}
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.IsActive = validate.Flag(req.IsActive, user.IsActive)
	user.Data = data

	if err := setPassword(user, req); err != nil {
		return nil, err
	}

	if into == nil {
		roles, err := h.store.DefaultUserRoles(ctx)
		if err != nil {
			return nil, err
		}

		user.Roles = roles
	}

	return user, nil
}

func setPassword(user *model.User, req *userRequest) error {
	if req.Password != "" {
		err := user.SetPassword(req.Password)
		if errors.Is(err, model.ErrPasswordTooLong) {
			return respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
		}
		if err != nil {
			return err
		}
	}

	temporary := validate.Flag(req.IsPasswordTemporary, user.IsPasswordTemporary)

	if temporary && !user.HasPassword {
		return respond.Fault{
			Status:  http.StatusBadRequest,
			Message: "is_password_temporary needs a password to be set",
		}
	}

	user.IsPasswordTemporary = temporary

	return nil
}

func visibleRoles(c *gin.Context, user *model.User) {
	user.Roles = slices.DeleteFunc(user.Roles, func(role model.UserRole) bool {
		return !session.SeesRole(c, role)
	})
}

func (h *Handler) find(c *gin.Context) (*model.User, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, respond.UserNotFound)
		return nil, false
	}

	user, err := h.store.User(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, respond.UserNotFound)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading user failed")
		return nil, false
	}

	return user, true
}
