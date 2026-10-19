// Package mailsettings manages how the server sends mail: the SMTP server, the
// text of each email, and a test send.
package mailsettings

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"loginer/i18n"
	"loginer/internal/api/kit/audit"
	"loginer/internal/api/kit/respond"
	"loginer/internal/brand"
	"loginer/internal/jose"
	"loginer/internal/mail"
	"loginer/internal/model"
	"loginer/internal/store"
)

type Handler struct {
	store  *store.Store
	sealer *jose.Sealer
	audit  audit.Recorder
	log    *slog.Logger
}

func New(st *store.Store, sealer *jose.Sealer, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, sealer: sealer, audit: recorder, log: log}
}

// Get returns the mail settings.
func (h *Handler) Get(c *gin.Context) {
	settings, err := h.store.MailSettings(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the mail settings failed")
		return
	}

	c.JSON(http.StatusOK, newResponse(*settings))
}

// Update changes the settings; omitted fields stay, including the password
// unless a new one is typed.
func (h *Handler) Update(c *gin.Context) {
	settings, err := h.store.MailSettings(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the mail settings failed")
		return
	}

	var req settingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	changed, err := req.applyTo(settings, h.sealer)
	if err != nil {
		respond.Failure(c, h.log, err, "validating the mail settings failed")
		return
	}

	if len(changed) == 0 {
		c.JSON(http.StatusOK, newResponse(*settings))
		return
	}

	if err := h.store.SaveMailSettings(c.Request.Context(), settings); err != nil {
		respond.Failure(c, h.log, err, "updating the mail settings failed")
		return
	}

	// Which settings moved, never what to: the password is one of them, and
	// the activity log is read by more people than this page is.
	h.audit.RecordWith(c, "mail.settings_updated", targetType, targetID, map[string]any{
		"fields": changed,
	})

	c.JSON(http.StatusOK, newResponse(*settings))
}

// Test sends one message with the form's settings, so a server can be tried
// before saving. Without a typed password the stored one is used.
func (h *Handler) Test(c *gin.Context) {
	var req testRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	stored, err := h.store.MailSettings(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "loading the mail settings failed")
		return
	}

	settings, err := req.settings(stored, h.sealer)
	if err != nil {
		respond.Failure(c, h.log, err, "reading the mail settings to test failed")
		return
	}

	to := req.To
	if to == "" {
		respond.Fail(c, testNeedsRecipient)
		return
	}

	err = mail.Deliver(c.Request.Context(), settings, mail.Message{
		To:      to,
		Subject: brand.Name + ": a test message",
		Body:    testBody,
	})
	if err != nil {
		// What the mail server said is the whole point of the test, so it is
		// given back rather than logged and hidden behind a 500.
		h.log.Warn("a test email failed", "error", err, "host", settings.Host)
		h.audit.RecordWith(c, "mail.test_failed", targetType, targetID, map[string]any{
			"to": to, "host": settings.Host,
		})

		respond.Fail(c, testFailed, "reason", err.Error())

		return
	}

	h.audit.RecordWith(c, "mail.test_sent", targetType, targetID, map[string]any{
		"to": to, "host": settings.Host,
	})

	c.JSON(http.StatusOK, gin.H{"sent": true, "to": to})
}

const testBody = "This is a test message from your " + brand.Name + " installation.\n\n" +
	"If you are reading it, the mail settings on the Mail page work: " +
	"password resets, address confirmations and one-time codes will reach people.\n"

// Content returns every email's text in each offered language, with English
// beside it.
func (h *Handler) Content(c *gin.Context) {
	ctx := c.Request.Context()

	languages, err := h.store.Languages(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "listing languages failed")
		return
	}

	english := i18n.Resolve(i18n.ID)

	out := make([]languageContent, 0, len(languages))
	for _, language := range languages {
		messages, err := h.store.Translation(ctx, language.ID, string(i18n.ID))
		if err != nil {
			respond.Failure(c, h.log, err, "loading a translation failed")
			return
		}

		out = append(out, newLanguageContent(language, messages, english))
	}

	c.JSON(http.StatusOK, contentResponse{Messages: model.MailMessageSpecs, Languages: out})
}

// SaveContent writes one language's email text, merging only the email keys.
// Any other key is refused as a client bug.
func (h *Handler) SaveContent(c *gin.Context) {
	language, err := h.store.Language(c.Request.Context(), c.Param("code"))
	if errors.Is(err, store.ErrNotFound) {
		respond.Fail(c, respond.LanguageNotFound)
		return
	}
	if err != nil {
		respond.Failure(c, h.log, err, "loading a language failed")
		return
	}

	var req contentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	if err := req.validate(); err != nil {
		respond.Failure(c, h.log, err, "validating email content failed")
		return
	}

	if err := h.store.SaveTranslationKeys(c.Request.Context(), language, string(i18n.ID), req.Messages); err != nil {
		respond.Failure(c, h.log, err, "saving email content failed")
		return
	}

	h.audit.RecordWith(c, "mail.content_updated", contentTargetType, language.Code, map[string]any{
		"language": language.Code, "keys": len(req.Messages),
	})

	messages, err := h.store.Translation(c.Request.Context(), language.ID, string(i18n.ID))
	if err != nil {
		respond.Failure(c, h.log, err, "loading a translation failed")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"language": newLanguageContent(*language, messages, i18n.Resolve(i18n.ID)),
	})
}
