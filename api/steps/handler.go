// Package steps takes the answers to the pages a sign-in passes once the
// user is known: the organization's terms, and what an application asks for.
package steps

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/api/kit/respond"
	"vard/api/kit/signedin"
	"vard/core/oidc"
)

type Handler struct {
	provider *oidc.Service
	log      *slog.Logger
}

func New(provider *oidc.Service, log *slog.Logger) *Handler {
	return &Handler{provider: provider, log: log}
}

// Consent is what the consent page lists for a sign-in under way.
func (h *Handler) Consent(c *gin.Context) {
	page, err := h.provider.ConsentFor(c.Request.Context(), c.Query("request"), signedin.Session(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "loading a consent page failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"consent": page})
}

// AnswerConsent takes the consent page's answer.
func (h *Handler) AnswerConsent(c *gin.Context) {
	h.answerStep(c, h.provider.AnswerConsent, "answering a consent page failed")
}

// AnswerAgreements takes the Agreements page's answer.
func (h *Handler) AnswerAgreements(c *gin.Context) {
	h.answerStep(c, h.provider.AnswerAgreements, "answering the agreements page failed")
}

type stepAnswer func(ctx context.Context, handle string, session *oidc.Session, yes bool, client oidc.Client) (string, error)

func (h *Handler) answerStep(c *gin.Context, answer stepAnswer, note string) {
	var req stepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, note)
		return
	}

	location, err := answer(c.Request.Context(), req.Request, signedin.Session(c), req.Yes, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, note)
		return
	}

	c.JSON(http.StatusOK, stepResponse{RedirectTo: location})
}

// Device is what a device the person is about to allow asks for.
func (h *Handler) Device(c *gin.Context) {
	page, err := h.provider.DeviceFor(c.Request.Context(), c.Query("user_code"), signedin.Session(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "loading a device failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": page})
}

// AnswerDevice allows or refuses a device for the signed-in person.
func (h *Handler) AnswerDevice(c *gin.Context) {
	var req deviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating a device answer failed")
		return
	}

	err := h.provider.AnswerDevice(c.Request.Context(), req.UserCode, signedin.Session(c), req.Yes, signedin.Client(c))
	if err != nil {
		signedin.Fail(c, h.log, err, "answering a device failed")
		return
	}

	c.Status(http.StatusNoContent)
}
