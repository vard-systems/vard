package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/audit"
	"loginer/internal/api/respond"
	"loginer/internal/api/session"
	authsvc "loginer/internal/auth"
	"loginer/internal/model"
	"loginer/internal/store"
)

var (
	wrongPassword   = respond.Define(http.StatusBadRequest, "admin_wrong_password", respond.Admin)
	avatarInvalid   = respond.Define(http.StatusBadRequest, "admin_avatar_invalid", respond.Admin)
	sessionCurrent  = respond.Define(http.StatusConflict, "admin_session_current", respond.Admin)
	sessionNotFound = respond.Define(http.StatusNotFound, "admin_session_not_found", respond.Admin)
)

const targetType = "admin_user"

type Handler struct {
	auth  *authsvc.Service
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger

	secure bool
}

// New returns a Handler.
func New(service *authsvc.Service, st *store.Store, recorder audit.Recorder, log *slog.Logger, secure bool) *Handler {
	return &Handler{auth: service, store: st, audit: recorder, log: log, secure: secure}
}

// Login checks the credentials and sets the session cookie.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a sign-in failed")
		return
	}

	token, admin, state, err := h.auth.Login(c.Request.Context(), req.Username, req.Password, session.RequestOf(c))
	switch {
	case errors.Is(err, authsvc.ErrInvalidCredentials):
		// One message for every kind of failure: which usernames exist is not
		// something a sign-in page should reveal.
		respond.Fail(c, wrongCredentials)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "login failed")
		return
	}

	session.Set(c, token, h.secure)

	// A right password with a second step still to go says which step; the
	// administrator is not shown until they are through it.
	if state != authsvc.StateSignedIn {
		c.JSON(http.StatusOK, gin.H{"next": state})
		return
	}

	c.JSON(http.StatusOK, gin.H{"admin": newAdminResponse(admin)})
}

// State reports how far this browser's session has got (password, code,
// enrolment). It needs no session.
func (h *Handler) State(c *gin.Context) {
	token := session.Token(c)

	_, _, state, err := h.auth.Session(c.Request.Context(), token)
	if err != nil && !errors.Is(err, authsvc.ErrNoSession) {
		respond.Failure(c, h.log, err, "reading the session failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"state": state, "mfa_required": h.auth.MFARequired(c.Request.Context())})
}

// VerifyMFA finishes a sign-in waiting for a second factor.
func (h *Handler) VerifyMFA(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	token := session.Token(c)

	admin, err := h.auth.VerifySignIn(c.Request.Context(), token, req.Code, session.RequestOf(c))
	switch {
	case errors.Is(err, authsvc.ErrNoSession):
		respond.Fail(c, signInExpired)
		return
	case errors.Is(err, authsvc.ErrInvalidCode):
		respond.Fail(c, signInCodeWrong)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "verifying a second factor failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"admin": newAdminResponse(admin)})
}

// Logout revokes the session and clears the cookie.
func (h *Handler) Logout(c *gin.Context) {
	token := session.Token(c)

	if err := h.auth.Logout(c.Request.Context(), token, session.RequestOf(c)); err != nil {
		h.log.Error("logout failed", "error", err)
	}

	session.Clear(c, h.secure)

	c.JSON(http.StatusOK, gin.H{"status": "signed out"})
}

// Me returns the signed-in administrator, and tells the browser whether its
// session is still alive. The organisation's public name and logo come with it
// for the panel's header.
func (h *Handler) Me(c *gin.Context) {
	organization, err := h.store.Organization(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the organization failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"admin":        newAdminResponse(session.Admin(c)),
		"organization": newOrganizationBrand(*organization),
	})
}

// UpdateMe changes the caller's own name and address, nothing else, so every
// administrator may use it.
func (h *Handler) UpdateMe(c *gin.Context) {
	var req profileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a profile failed")
		return
	}

	admin := session.Admin(c)
	profile := authsvc.Profile{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		AvatarURL: req.AvatarURL,
	}

	err := h.auth.UpdateProfile(c.Request.Context(), admin, profile, req.CurrentPassword)
	switch {
	case errors.Is(err, authsvc.ErrWrongPassword):
		respond.Fail(c, wrongPassword)
		return
	case errors.Is(err, store.ErrDuplicate):
		respond.Fail(c, respond.AdminEmailTaken)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "updating a profile failed")
		return
	}

	h.audit.Record(c, "admin.profile_updated", targetType, admin.ID.String())

	c.JSON(http.StatusOK, gin.H{"admin": newAdminResponse(admin)})
}

// ChangePassword sets the caller's own password, given the one they have.
// Every other session they have open ends; the one they are using stays.
func (h *Handler) ChangePassword(c *gin.Context) {
	var req passwordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a password change failed")
		return
	}

	admin := session.Admin(c)

	err := h.auth.ChangePassword(c.Request.Context(), admin, session.ID(c), req.CurrentPassword, req.NewPassword)
	switch {
	case errors.Is(err, authsvc.ErrWrongPassword):
		respond.Fail(c, wrongPassword)
		return
	case errors.Is(err, model.ErrPasswordTooLong):
		respond.Fail(c, respond.AdminPasswordTooLong)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "changing a password failed")
		return
	}

	h.audit.Record(c, "admin.password_changed", targetType, admin.ID.String())

	c.JSON(http.StatusOK, gin.H{"status": "changed"})
}

// Sessions lists the caller's own sessions, so they can see where they are
// signed in.
func (h *Handler) Sessions(c *gin.Context) {
	sessions, err := h.store.AdminSessionsFor(c.Request.Context(), session.Admin(c).ID, 20)
	if err != nil {
		respond.Failure(c, h.log, err, "listing sessions failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": newSessionResponses(sessions, session.ID(c))})
}

// EndSession signs one of the administrator's other browsers out. Another
// administrator's session is answered as not found.
func (h *Handler) EndSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, sessionNotFound)
		return
	}

	if id == session.ID(c) {
		respond.Fail(c, sessionCurrent)
		return
	}

	admin := session.Admin(c)
	ended, err := h.store.RevokeOwnAdminSession(c.Request.Context(), admin.ID, id, time.Now())
	if err != nil {
		respond.Failure(c, h.log, err, "ending a session failed")
		return
	}
	if !ended {
		respond.Fail(c, sessionNotFound)
		return
	}

	h.audit.Record(c, "admin.session_ended", targetType, admin.ID.String())
	c.Status(http.StatusNoContent)
}

// EndOtherSessions signs the administrator out everywhere but here: what to
// do after using a shared computer, or losing a laptop.
func (h *Handler) EndOtherSessions(c *gin.Context) {
	admin := session.Admin(c)

	ended, err := h.store.RevokeOtherAdminSessionsFor(c.Request.Context(), admin.ID, session.ID(c), time.Now())
	if err != nil {
		respond.Failure(c, h.log, err, "ending the other sessions failed")
		return
	}

	if ended > 0 {
		h.audit.RecordWith(c, "admin.sessions_ended", targetType, admin.ID.String(), map[string]any{
			"count": ended,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ended": ended})
}
