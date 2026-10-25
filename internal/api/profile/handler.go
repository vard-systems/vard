// Package profile answers a signed-in person about their own account: their
// name, address and password, where they are signed in, and the applications
// that can act for them. Signing in is the account package's.
package profile

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/signedin"
	"vard/internal/oidc"
)

type Handler struct {
	provider *oidc.Service
	log      *slog.Logger
}

func New(provider *oidc.Service, log *slog.Logger) *Handler {
	return &Handler{provider: provider, log: log}
}

// Me returns the signed-in user.
func (h *Handler) Me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"user": newUserResponse(signedin.Session(c).User)})
}

// UpdateMe changes the signed-in user's name.
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

	user, err := h.provider.UpdateProfile(c.Request.Context(), signedin.Session(c), oidc.Profile{
		FirstName: req.FirstName,
		LastName:  req.LastName,
	}, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "updating a profile failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": newUserResponse(user)})
}

// ChangePassword replaces the signed-in user's password.
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

	err := h.provider.ChangePassword(c.Request.Context(), signedin.Session(c), req.CurrentPassword, req.NewPassword, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "changing a password failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "changed"})
}

// ChangeEmail starts moving the signed-in user to another address, given their
// current password. It answers the same whether or not the address is taken,
// and only to a browser session: an access token is refused.
func (h *Handler) ChangeEmail(c *gin.Context) {
	var req emailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating an address change failed")
		return
	}

	if err := h.provider.RequestEmailChange(c.Request.Context(), signedin.Session(c), req.Email, req.CurrentPassword, signedin.Client(c)); err != nil {
		signedin.Fail(c, h.log, err, "starting an address change failed")
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"status": "sent"})
}

// Sessions lists where the signed-in user is signed in.
func (h *Handler) Sessions(c *gin.Context) {
	sessions, err := h.provider.Sessions(c.Request.Context(), signedin.Session(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "listing sessions failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"sessions": sessions})
}

// EndSession signs one of the user's other browsers out.
func (h *Handler) EndSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, signedin.Problem(oidc.ErrNotYours))
		return
	}

	if err := h.provider.EndSession(c.Request.Context(), signedin.Session(c), id, signedin.Client(c)); err != nil {
		signedin.Fail(c, h.log, err, "ending a session failed")
		return
	}

	c.Status(http.StatusNoContent)
}

// Applications lists the applications that can still act for the user.
func (h *Handler) Applications(c *gin.Context) {
	apps, err := h.provider.ConnectedApplications(c.Request.Context(), signedin.Session(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "listing connected applications failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"applications": apps})
}

// Disconnect revokes an application's refresh tokens for the user.
func (h *Handler) Disconnect(c *gin.Context) {
	err := h.provider.DisconnectApplication(c.Request.Context(), signedin.Session(c), c.Param("client_id"), signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "disconnecting an application failed")
		return
	}

	c.Status(http.StatusNoContent)
}
