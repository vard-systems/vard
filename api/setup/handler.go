// Package setup reports whether an administrator exists and creates the first
// one. Both endpoints are open; the store refuses creating once anyone exists.
package setup

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"vard/api/kit/respond"
	"vard/core/auth"
	"vard/core/model"
	"vard/core/store"
)

type Handler struct {
	store *store.Store
	auth  *auth.Service
	log   *slog.Logger
}

func New(st *store.Store, service *auth.Service, log *slog.Logger) *Handler {
	return &Handler{store: st, auth: service, log: log}
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

	// Once the panel is set up, a setup is refused before it costs a
	// password hash and a breach lookup; CreateFirstAdmin still decides
	// under its lock, for setups that race.
	exists, err := h.store.AdminsExist(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "checking for administrators failed")
		return
	}
	if exists {
		respond.Fail(c, alreadySetUp)
		return
	}

	admin := model.Admin{
		Username:  req.Email,
		Email:     req.Email,
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Status:    model.StatusActive,
	}
	if h.auth.PasswordBreached(c.Request.Context(), req.Password) {
		respond.Fail(c, respond.AdminPasswordBreached)
		return
	}
	if err := admin.SetPassword(req.Password); errors.Is(err, model.ErrPasswordTooLong) {
		respond.Fail(c, respond.AdminPasswordTooLong)
		return
	} else if err != nil {
		respond.Failure(c, h.log, err, "hashing the password failed")
		return
	}

	err = h.store.CreateFirstAdmin(c.Request.Context(), &admin)
	switch {
	case errors.Is(err, store.ErrAdminExists):
		respond.Fail(c, alreadySetUp)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "creating the first administrator failed")
		return
	}

	h.log.Info("the first administrator was created", "email", admin.Email)

	c.JSON(http.StatusCreated, gin.H{"admin": newAdminResponse(&admin)})
}
