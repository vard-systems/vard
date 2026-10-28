// Package setup reports whether an administrator exists and creates the first
// one. Both endpoints are open, but creating takes the one-time token the
// server logged at startup, and the store refuses it once anyone exists.
package setup

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"vard/internal/api/kit/respond"
	"vard/internal/model"
	"vard/internal/store"
)

type Handler struct {
	store *store.Store
	log   *slog.Logger
}

func New(st *store.Store, log *slog.Logger) *Handler {
	return &Handler{store: st, log: log}
}

// Status says whether the panel still needs its first administrator.
func (h *Handler) Status(c *gin.Context) {
	exists, err := h.store.AdminsExist(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "checking for administrators failed")
		return
	}

	c.JSON(http.StatusOK, statusResponse{Required: !exists})
}

// Create makes the first administrator: a super admin, with the address and
// password whoever is setting the panel up chose.
func (h *Handler) Create(c *gin.Context) {
	var req setupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating the first administrator failed")
		return
	}

	admin := model.Admin{
		Username:  req.Email,
		Email:     req.Email,
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Status:    model.StatusActive,
	}
	if err := admin.SetPassword(req.Password); errors.Is(err, model.ErrPasswordTooLong) {
		respond.Fail(c, respond.AdminPasswordTooLong)
		return
	} else if err != nil {
		respond.Failure(c, h.log, err, "hashing the password failed")
		return
	}

	err := h.store.CreateFirstAdmin(c.Request.Context(), &admin, model.HashSecret(req.SetupToken), time.Now())
	switch {
	case errors.Is(err, store.ErrAdminExists):
		respond.Fail(c, alreadySetUp)
		return
	case errors.Is(err, store.ErrSetupTokenInvalid):
		respond.Fail(c, tokenRefused)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "creating the first administrator failed")
		return
	}

	h.log.Info("the first administrator was created", "email", admin.Email)

	c.JSON(http.StatusCreated, gin.H{"admin": newAdminResponse(&admin)})
}
