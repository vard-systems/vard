// Package mfa lets an administrator set up, confirm and turn off their
// authenticator, and make new recovery codes.
package mfa

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/internal/api/kit/respond"
	"vard/internal/api/kit/session"
	"vard/internal/auth"
)

type Handler struct {
	auth *auth.Service
	log  *slog.Logger
}

func New(service *auth.Service, log *slog.Logger) *Handler {
	return &Handler{auth: service, log: log}
}

// Status describes the administrator's second factor.
func (h *Handler) Status(c *gin.Context) {
	status, err := h.auth.Status(c.Request.Context(), session.Admin(c))
	if err != nil {
		respond.Failure(c, h.log, err, "reading two-factor status failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"mfa": status})
}

// Begin starts setting up an authenticator app, and answers the secret and
// the otpauth URI to show as a QR code.
func (h *Handler) Begin(c *gin.Context) {
	var req optionalCodeRequest
	if err := bindOptional(c, &req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	enrolment, err := h.auth.BeginTOTP(c.Request.Context(), session.Admin(c), req.Code, session.RequestOf(c))
	if err != nil {
		h.fail(c, err, "starting an authenticator set-up failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"enrolment": enrolment})
}

// Confirm finishes setting up with a code from the app, and answers the
// recovery codes — the only time they are shown.
func (h *Handler) Confirm(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	token := session.Token(c)

	codes, err := h.auth.ConfirmTOTP(c.Request.Context(), session.Admin(c), token, req.Code, session.RequestOf(c))
	if err != nil {
		h.fail(c, err, "confirming an authenticator failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"recovery_codes": codes})
}

// Disable turns two-factor sign-in off.
func (h *Handler) Disable(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	token := session.Token(c)

	if err := h.auth.DisableTOTP(c.Request.Context(), session.Admin(c), token, req.Code, session.RequestOf(c)); err != nil {
		h.fail(c, err, "turning two-factor sign-in off failed")
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) RecoveryCodes(c *gin.Context) {
	var req codeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	codes, err := h.auth.RegenerateRecoveryCodes(c.Request.Context(), session.Admin(c), req.Code, session.RequestOf(c))
	if err != nil {
		h.fail(c, err, "regenerating recovery codes failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"recovery_codes": codes})
}

func (h *Handler) fail(c *gin.Context, err error, note string) {
	switch {
	case errors.Is(err, auth.ErrInvalidCode):
		respond.Fail(c, codeWrong)
	case errors.Is(err, auth.ErrNotEnrolling):
		respond.Fail(c, notEnrolling)
	case errors.Is(err, auth.ErrMFADisabled):
		respond.Fail(c, alreadyOff)
	case errors.Is(err, auth.ErrMFAEnabled):
		respond.Fail(c, alreadyOn)
	case errors.Is(err, auth.ErrMFARequired):
		respond.Fail(c, mustStayOn)
	default:
		respond.Failure(c, h.log, err, note)
	}
}
