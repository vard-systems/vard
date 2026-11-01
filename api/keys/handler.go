// Package keys lists the token signing keys and rotates them on demand.
package keys

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/api/kit/audit"
	"vard/api/kit/respond"
	"vard/core/oidc"
)

type Handler struct {
	provider *oidc.Service
	audit    audit.Recorder
	log      *slog.Logger
}

func New(provider *oidc.Service, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{provider: provider, audit: recorder, log: log}
}

// List describes every published key: the one signing, the next one waiting,
// and the retired ones still published.
func (h *Handler) List(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"keys": h.provider.SigningKeys()})
}

// Rotate makes new keys now.
func (h *Handler) Rotate(c *gin.Context) {
	var req rotateRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			respond.Fail(c, respond.InvalidBody)
			return
		}
	}

	if err := h.provider.RotateKeys(c.Request.Context(), req.Immediate, req.RevokeOld); err != nil {
		respond.Failure(c, h.log, err, "rotating the signing keys failed")
		return
	}

	h.audit.RecordWith(c, "signing_keys.rotated", "", "", map[string]any{
		"immediate":  req.Immediate || req.RevokeOld,
		"revoke_old": req.RevokeOld,
	})

	c.JSON(http.StatusOK, gin.H{"keys": h.provider.SigningKeys()})
}
