// Package twostep answers a person about their own two-step verification:
// what they have, adding an authenticator or a passkey, removing one, and new
// recovery codes. Asking for a second factor at sign-in is the account
// package's, with the rest of signing in.
package twostep

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"loginer/internal/api/kit/respond"
	"loginer/internal/api/kit/signedin"
	"loginer/internal/oidc"
)

type Handler struct {
	provider *oidc.Service
	log      *slog.Logger
}

func New(provider *oidc.Service, log *slog.Logger) *Handler {
	return &Handler{provider: provider, log: log}
}

// TwoStep is the user's two-step verification, as the account page shows it.
func (h *Handler) TwoStep(c *gin.Context) {
	status, err := h.provider.TwoStep(c.Request.Context(), signedin.Session(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "loading two-step verification failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"two_step": status})
}

func (h *Handler) BeginAuthenticator(c *gin.Context) {
	var req presenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating an authenticator start failed")
		return
	}

	setup, err := h.provider.BeginAuthenticator(c.Request.Context(), signedin.Session(c), req.CurrentPassword, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "starting an authenticator failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"authenticator": setup})
}

func (h *Handler) ConfirmAuthenticator(c *gin.Context) {
	var req confirmAuthenticatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating an authenticator code failed")
		return
	}

	codes, err := h.provider.ConfirmAuthenticator(c.Request.Context(), signedin.Session(c), req.Code, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "confirming an authenticator failed")
		return
	}

	c.JSON(http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

func (h *Handler) BeginPasskey(c *gin.Context) {
	var req presenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a passkey start failed")
		return
	}

	options, err := h.provider.BeginPasskey(c.Request.Context(), signedin.Session(c), req.CurrentPassword, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "starting a passkey failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"passkey": options})
}

func (h *Handler) FinishPasskey(c *gin.Context) {
	var req finishPasskeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a passkey failed")
		return
	}

	codes, err := h.provider.FinishPasskey(c.Request.Context(), signedin.Session(c), req.Label, req.Response, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "adding a passkey failed")
		return
	}

	c.JSON(http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}

func (h *Handler) RemoveFactor(c *gin.Context) {
	var req presenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a factor removal failed")
		return
	}

	err := h.provider.RemoveFactor(c.Request.Context(), signedin.Session(c), c.Param("id"), req.CurrentPassword, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "removing a factor failed")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) NewRecoveryCodes(c *gin.Context) {
	var req presenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating new recovery codes failed")
		return
	}

	codes, err := h.provider.NewRecoveryCodes(c.Request.Context(), signedin.Session(c), req.CurrentPassword, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "replacing recovery codes failed")
		return
	}

	c.JSON(http.StatusOK, recoveryCodesResponse{RecoveryCodes: codes})
}
