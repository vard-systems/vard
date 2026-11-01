// Package languages manages the languages the sign-in pages are offered in, and
// their translations.
package languages

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/api/kit/audit"
	"vard/api/kit/respond"
	"vard/api/kit/session"
	"vard/api/kit/validate"
	"vard/core/model"
	"vard/core/store"
	"vard/i18n"
)

type Handler struct {
	store *store.Store
	audit audit.Recorder
	log   *slog.Logger
}

func New(st *store.Store, recorder audit.Recorder, log *slog.Logger) *Handler {
	return &Handler{store: st, audit: recorder, log: log}
}

// List returns every language, with how much of each app it translates.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	if _, err := h.store.DefaultLanguage(ctx); err != nil {
		respond.Failure(c, h.log, err, "loading the default language failed")
		return
	}

	languages, err := h.store.Languages(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "listing languages failed")
		return
	}

	ids := make([]uuid.UUID, 0, len(languages))
	for _, language := range languages {
		ids = append(ids, language.ID)
	}

	text, err := h.store.Translations(ctx, ids...)
	if err != nil {
		respond.Failure(c, h.log, err, "loading translations failed")
		return
	}

	c.JSON(http.StatusOK, newListResponse(languages, text))
}

// Create adds a language, with the text it starts from (createRequest).
func (h *Handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	text, err := h.startingText(c, strings.TrimSpace(req.CopyFrom))
	if err != nil {
		respond.Failure(c, h.log, err, "loading the text a language starts from failed")
		return
	}

	language, err := req.newLanguage()
	if err != nil {
		respond.Failure(c, h.log, err, "validating a language failed")
		return
	}

	language.Position = h.store.NextLanguagePosition(ctx)

	if err := h.store.CreateLanguage(ctx, language, text); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			respond.Fail(c, codeTaken)
			return
		}

		respond.Failure(c, h.log, err, "creating a language failed")
		return
	}

	h.audit.RecordWith(c, "language.created", targetType, language.ID.String(), map[string]any{
		"language": language.Code, "copied_from": req.CopyFrom,
	})

	h.answer(c, http.StatusCreated, language)
}

// startingText is the text of the installed language a new one copies, every
// key filled in, or nothing when it starts untranslated.
func (h *Handler) startingText(c *gin.Context, from string) (map[string]map[string]string, error) {
	if from == "" {
		return nil, nil
	}

	ctx := c.Request.Context()

	source, err := h.store.Language(ctx, from)
	switch {
	case errors.Is(err, store.ErrNotFound):
		return nil, nothingToCopy.With("code", from)
	case err != nil:
		return nil, err
	}

	text := make(map[string]map[string]string, len(i18n.Apps))
	for _, app := range i18n.Apps {
		if text[string(app)], err = h.store.ResolvedTranslation(ctx, source, app); err != nil {
			return nil, err
		}
	}

	return text, nil
}

// Update changes a language's names and whether and where it is offered.
func (h *Handler) Update(c *gin.Context) {
	language, ok := h.find(c)
	if !ok {
		return
	}

	var req languageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	wasDefault := language.IsDefault

	if err := req.applyTo(language); err != nil {
		respond.Failure(c, h.log, err, "validating a language failed")
		return
	}

	// The default is what every page is drawn in before somebody chooses, so
	// it is moved to another language rather than simply taken away.
	if wasDefault && !language.IsDefault {
		respond.Fail(c, keepADefault)
		return
	}

	if err := h.store.SaveLanguage(c.Request.Context(), language); err != nil {
		respond.Failure(c, h.log, err, "updating a language failed")
		return
	}

	h.audit.RecordWith(c, "language.updated", targetType, language.ID.String(), map[string]any{
		"language": language.Code, "is_enabled": language.IsEnabled, "is_default": language.IsDefault,
	})

	h.answer(c, http.StatusOK, language)
}

// Delete removes a language and its text. Anybody who had chosen it is shown
// the default from their next page on.
func (h *Handler) Delete(c *gin.Context) {
	language, ok := h.find(c)
	if !ok {
		return
	}

	err := h.store.DeleteLanguage(c.Request.Context(), language)
	switch {
	case errors.Is(err, store.ErrProtectedLanguage) && language.Code == model.BaseLanguage:
		respond.Fail(c, baseStays)
		return
	case errors.Is(err, store.ErrProtectedLanguage):
		respond.Fail(c, defaultStays)
		return
	case err != nil:
		respond.Failure(c, h.log, err, "deleting a language failed")
		return
	}

	h.audit.RecordWith(c, "language.deleted", targetType, language.ID.String(), map[string]any{
		"language": language.Code,
	})

	c.Status(http.StatusNoContent)
}

// Translation returns one language's text for one app, for the editor.
func (h *Handler) Translation(c *gin.Context) {
	language, app, ok := h.findWithApp(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()

	messages, err := h.store.Translation(ctx, language.ID, string(app))
	if err != nil {
		respond.Failure(c, h.log, err, "loading a translation failed")
		return
	}

	base, err := h.store.Language(ctx, model.BaseLanguage)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		respond.Failure(c, h.log, err, "loading the base language failed")
		return
	}

	english := i18n.Resolve(app)
	if base != nil {
		if english, err = h.store.ResolvedTranslation(ctx, base, app); err != nil {
			respond.Failure(c, h.log, err, "loading the base language failed")
			return
		}
	}

	c.JSON(http.StatusOK, translationResponse{
		App:      app,
		Keys:     i18n.Keys(app),
		Base:     english,
		Messages: messages,
	})
}

// SaveTranslation replaces one language's text for one app. Unknown keys are
// dropped and counted, so older files still import.
func (h *Handler) SaveTranslation(c *gin.Context) {
	language, app, ok := h.findWithApp(c)
	if !ok {
		return
	}

	var req translationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := validate.Struct(req); err != nil {
		respond.Failure(c, h.log, err, "validating a translation failed")
		return
	}

	messages, ignored := i18n.Known(app, req.Messages)

	// Email text is super-admin only, since the reset email decides where its
	// live link goes. A translator's save keeps the stored email text instead
	// of failing, because the editor sends the whole draft.
	if !session.Admin(c).IsSuperAdmin() {
		stored, err := h.store.Translation(c.Request.Context(), language.ID, string(app))
		if err != nil {
			respond.Failure(c, h.log, err, "loading a translation failed")
			return
		}
		for key := range messages {
			if !model.IsMailTextKey(key) {
				continue
			}
			if value, ok := stored[key]; ok {
				messages[key] = value
			} else {
				delete(messages, key)
			}
		}
	}

	if key := model.TooLongMessage(messages); key != "" {
		respond.Fail(c, translationTooLong, "key", key, "max", model.MaxMessageLength)
		return
	}

	if err := h.store.SaveTranslation(c.Request.Context(), language, string(app), messages); err != nil {
		respond.Failure(c, h.log, err, "saving a translation failed")
		return
	}

	h.audit.RecordWith(c, "language.translated", targetType, language.ID.String(), map[string]any{
		"language": language.Code, "app": string(app), "keys": len(messages), "ignored": ignored,
	})

	one, err := h.describe(c, language)
	if err != nil {
		respond.Failure(c, h.log, err, "loading translations failed")
		return
	}

	c.JSON(http.StatusOK, savedResponse{Language: one, Ignored: ignored})
}

func (h *Handler) find(c *gin.Context) (*model.Language, bool) {
	language, err := h.store.Language(c.Request.Context(), c.Param("code"))

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, respond.LanguageNotFound)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading a language failed")
		return nil, false
	}

	return language, true
}

func (h *Handler) findWithApp(c *gin.Context) (*model.Language, i18n.App, bool) {
	app, known := i18n.ParseApp(c.Param("app"))
	if !known {
		respond.Fail(c, noSuchApp)
		return nil, "", false
	}

	language, ok := h.find(c)

	return language, app, ok
}

func (h *Handler) answer(c *gin.Context, status int, language *model.Language) {
	one, err := h.describe(c, language)
	if err != nil {
		respond.Failure(c, h.log, err, "loading translations failed")
		return
	}

	c.JSON(status, response{Language: one})
}

func (h *Handler) describe(c *gin.Context, language *model.Language) (languageResponse, error) {
	text, err := h.store.Translations(c.Request.Context(), language.ID)
	if err != nil {
		return languageResponse{}, err
	}

	return newLanguageResponse(*language, text[language.ID]), nil
}
