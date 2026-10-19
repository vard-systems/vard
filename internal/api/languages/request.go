package languages

import (
	"net/http"

	"loginer/internal/api/kit/respond"
	"loginer/internal/api/kit/validate"
	"loginer/internal/model"
)

// targetType is what a language is called in the activity log.
const targetType = "language"

// What these endpoints refuse, as the Languages page shows it.
var (
	codeTaken          = respond.Define(http.StatusConflict, "language_code_taken", respond.Admin)
	nothingToCopy      = respond.Define(http.StatusBadRequest, "language_copy_missing", respond.Admin)
	keepADefault       = respond.Define(http.StatusBadRequest, "language_default_required", respond.Admin)
	defaultStays       = respond.Define(http.StatusBadRequest, "language_default_protected", respond.Admin)
	baseStays          = respond.Define(http.StatusBadRequest, "language_base_protected", respond.Admin)
	noSuchApp          = respond.Define(http.StatusNotFound, "translation_app_not_found", respond.Admin)
	translationTooLong = respond.Define(http.StatusBadRequest, "translation_too_long", respond.Admin)
)

func init() {
	validate.Register("languagetag", func(value string) bool {
		return (model.Language{Code: value, Name: "-", NativeName: "-"}).Validate() == nil
	})
}

// createRequest adds a language. CopyFrom is another installed language to
// copy, a shipped language to restore, or empty to start untranslated.
type createRequest struct {
	Code       string `json:"code" validate:"required,max=16,languagetag"`
	Name       string `json:"name" validate:"required,max=64"`
	NativeName string `json:"native_name" validate:"required,max=64"`
	IsEnabled  *bool  `json:"is_enabled"`
	IsDefault  *bool  `json:"is_default"`
	CopyFrom   string `json:"copy_from"`
}

// languageRequest is a PATCH: nil leaves a setting alone. The code is in the
// path and cannot change.
type languageRequest struct {
	Name       *string `json:"name" validate:"omitnil,min=1,max=64"`
	NativeName *string `json:"native_name" validate:"omitnil,min=1,max=64"`
	IsEnabled  *bool   `json:"is_enabled"`
	IsDefault  *bool   `json:"is_default"`
	Position   *int    `json:"position" validate:"omitnil,min=0"`
}

// translationRequest replaces one language's text for one app. `$name` and
// `$native` in an imported file are ignored.
type translationRequest struct {
	Messages map[string]string `json:"messages" validate:"required"`
}
