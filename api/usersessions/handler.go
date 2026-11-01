// Package usersessions lists users' sign-in sessions and ends them.
package usersessions

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/internal/infrastructure/store"
	"vard/internal/interfaces/api/kit/audit"
	"vard/internal/interfaces/api/kit/query"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/validate"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger
	now   func() time.Time
}

func New(st *store.Store, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, audit: recorder, log: log, now: time.Now}
}

// List returns a page of active sessions, newest first.
func (h *Handler) List(c *gin.Context) {
	var req listRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := validate.Struct(req); err != nil {
		respond.Failure(c, h.log, err, "validating a sessions query failed")
		return
	}

	limit := query.Int(c, "limit", pageSize, pageMax)

	found, err := h.store.ActiveSessions(c.Request.Context(), req.query(limit+1), h.now())
	if err != nil {
		respond.Failure(c, h.log, err, "listing sessions failed")
		return
	}

	out := listResponse{Sessions: make([]sessionResponse, 0, min(len(found), limit))}
	for i, session := range found {
		if i == limit {
			out.Next = found[i-1].ID.String()
			break
		}
		out.Sessions = append(out.Sessions, newSessionResponse(session))
	}

	c.JSON(http.StatusOK, out)
}

// End signs one session out. The user's applications keep their tokens: it
// is the browser that is signed out, as when the user ends it themselves.
func (h *Handler) End(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, sessionAbsent)
		return
	}

	ctx := c.Request.Context()

	session, err := h.store.UserSession(ctx, id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, sessionAbsent)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "loading a session failed")
		return
	}

	if err := h.store.RevokeUserSession(ctx, session.ID, h.now()); err != nil {
		respond.Failure(c, h.log, err, "ending a session failed")
		return
	}

	h.audit.RecordWith(c, "user.session_ended", targetType, session.UserID.String(), map[string]any{
		"ip": session.IP,
	})

	c.Status(http.StatusNoContent)
}

// SignOutUser ends every session a user has and revokes every refresh token
// their applications hold, so they are signed out everywhere at once.
func (h *Handler) SignOutUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, respond.NotFoundAny)
		return
	}

	ctx := c.Request.Context()

	user, err := h.store.User(ctx, id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, respond.NotFoundAny)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "loading a user failed")
		return
	}

	sessions, tokens, err := h.store.SignOutUser(ctx, user.ID, h.now())
	if err != nil {
		respond.Failure(c, h.log, err, "signing a user out failed")
		return
	}

	h.audit.RecordWith(c, "user.signed_out_everywhere", targetType, user.ID.String(), map[string]any{
		"sessions": sessions, "tokens": tokens,
	})

	c.JSON(http.StatusOK, gin.H{"sessions": sessions, "tokens": tokens})
}
