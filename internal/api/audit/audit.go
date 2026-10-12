package audit

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/session"
	"loginer/internal/model"
	"loginer/internal/store"
)

type Recorder struct {
	store *store.Store
	log   *slog.Logger
}

func New(st *store.Store, log *slog.Logger) Recorder {
	return Recorder{store: st, log: log}
}

func (r Recorder) Record(c *gin.Context, action, targetType, targetID string) {
	r.RecordWith(c, action, targetType, targetID, nil)
}

// RecordWith is Record with related metadata, so the entry can be found from
// there too. Never include secrets.
func (r Recorder) RecordWith(c *gin.Context, action, targetType, targetID string, metadata map[string]any) {
	actor := session.Admin(c)
	if actor == nil {
		return
	}

	// An application calling with a token is no administrator in the table:
	// the entry names it by its client ID, and points at nobody.
	var adminID *uuid.UUID
	if !actor.Service {
		id := actor.ID
		adminID = &id
	}

	entry := model.AuditLog{
		AdminID:    adminID,
		ActorEmail: actor.Username,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Metadata:   metadata,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	}

	if err := r.store.WriteAudit(c.Request.Context(), &entry); err != nil {
		r.log.Error("writing the activity log failed", "error", err, "action", action)
	}
}
