// Package fields manages the custom profile fields users carry.
package fields

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/kit/audit"
	"loginer/internal/api/kit/respond"
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

// List returns the fields a user record has, in the order they are shown. The
// panel builds both its table and its form from this.
func (h *Handler) List(c *gin.Context) {
	fields, err := h.store.UserFields(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "listing user fields failed")
		return
	}

	c.JSON(http.StatusOK, newListResponse(fields))
}

// Create adds a field to every user record. Existing users simply have no
// value for it until they are edited.
func (h *Handler) Create(c *gin.Context) {
	var req fieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	field, err := req.newField()
	if err != nil {
		respond.Failure(c, h.log, err, "validating a field failed")
		return
	}

	field.Position = h.store.NextFieldPosition(c.Request.Context())

	if err := h.store.CreateUserField(c.Request.Context(), field); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			respond.Fail(c, nameTaken)
			return
		}

		respond.Failure(c, h.log, err, "creating user field failed")
		return
	}

	h.audit.Record(c, "user_field.created", targetType, field.Name)

	c.JSON(http.StatusCreated, gin.H{"field": newFieldResponse(*field)})
}

// Update changes a field's rules. Name and type never change, since records
// hold values under them.
func (h *Handler) Update(c *gin.Context) {
	field, ok := h.find(c)
	if !ok {
		return
	}

	var req rulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if err := req.applyTo(field); err != nil {
		respond.Failure(c, h.log, err, "validating a field failed")
		return
	}

	if err := h.store.SaveUserField(c.Request.Context(), field); err != nil {
		respond.Failure(c, h.log, err, "updating user field failed")
		return
	}

	h.audit.Record(c, "user_field.updated", targetType, field.Name)

	c.JSON(http.StatusOK, gin.H{"field": newFieldResponse(*field)})
}

// Delete removes a field; stored values are dropped when each user is next
// saved.
func (h *Handler) Delete(c *gin.Context) {
	field, ok := h.find(c)
	if !ok {
		return
	}

	if err := h.store.DeleteUserField(c.Request.Context(), field); err != nil {
		respond.Failure(c, h.log, err, "deleting user field failed")
		return
	}

	h.audit.Record(c, "user_field.deleted", targetType, field.Name)

	c.Status(http.StatusNoContent)
}

func (h *Handler) find(c *gin.Context) (*model.UserField, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, fieldAbsent)
		return nil, false
	}

	field, err := h.store.UserField(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, fieldAbsent)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading user field failed")
		return nil, false
	}

	return field, true
}
