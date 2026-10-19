package languages

import (
	"time"

	"github.com/google/uuid"

	"loginer/i18n"
	"loginer/internal/model"
)

// languageResponse is one language as the panel sees it: its settings, and
// how much of each app it translates.
type languageResponse struct {
	Code string `json:"code"`
	// Name is what the language is called in English, Native what it calls
	// itself.
	Name       string `json:"name"`
	NativeName string `json:"native_name"`

	// Apps are the apps this language is translated for, which today is the
	// sign-in pages and nothing else.
	Apps []i18n.App `json:"apps"`

	// Coverage and Missing are per app: the percentage of base keys translated
	// and how many are missing.
	Coverage map[i18n.App]int `json:"coverage"`
	Missing  map[i18n.App]int `json:"missing"`

	IsEnabled bool `json:"is_enabled"`
	IsDefault bool `json:"is_default"`
	Position  int  `json:"position"`

	// Base marks the language every other is a translation of: it cannot be
	// turned off or removed.
	Base bool `json:"base"`

	// Shipped says the server ships a translation of this language, so a key
	// a release adds reaches it on the next start.
	Shipped bool `json:"shipped"`

	UpdatedAt time.Time `json:"updated_at"`
}

// shippedResponse is a shipped language this installation lacks, which the
// panel offers to add.
type shippedResponse struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
}

// listResponse is what the list endpoint answers with.
type listResponse struct {
	Languages []languageResponse `json:"languages"`
	// Apps are the two catalogs a language is counted against, so the panel
	// names the columns from the server rather than from a list of its own.
	Apps    []i18n.App        `json:"apps"`
	Shipped []shippedResponse `json:"shipped"`
}

type response struct {
	Language languageResponse `json:"language"`
}

// translationResponse is one language's text for one app with the base text
// beside each key.
type translationResponse struct {
	App  i18n.App `json:"app"`
	Keys []string `json:"keys"`
	// Base is the text the base language shows, every key filled in: what
	// the translator is translating, and what a missing key falls back to.
	Base     map[string]string `json:"base"`
	Messages map[string]string `json:"messages"`
}

// savedResponse is the language after saving and how many sent keys were
// unknown.
type savedResponse struct {
	Language languageResponse `json:"language"`
	Ignored  int              `json:"ignored"`
}

func newLanguageResponse(language model.Language, text map[string]map[string]string, shipped map[string]bool) languageResponse {
	out := languageResponse{
		Code:       language.Code,
		Name:       language.Name,
		NativeName: language.NativeName,
		Apps:       i18n.AppsFor(language.Code),
		Coverage:   map[i18n.App]int{},
		Missing:    map[i18n.App]int{},
		IsEnabled:  language.IsEnabled,
		IsDefault:  language.IsDefault,
		Position:   language.Position,
		Base:       language.Code == model.BaseLanguage,
		Shipped:    shipped[language.Code],
		UpdatedAt:  language.UpdatedAt,
	}

	for _, app := range out.Apps {
		out.Coverage[app], out.Missing[app] = i18n.Coverage(app, text[string(app)])
	}

	return out
}

func newListResponse(languages []model.Language, text map[uuid.UUID]map[string]map[string]string, files []i18n.File) listResponse {
	shipped := make(map[string]bool, len(files))
	for _, file := range files {
		shipped[file.Code] = true
	}

	out := listResponse{
		Languages: make([]languageResponse, 0, len(languages)),
		Apps:      i18n.Apps,
		Shipped:   []shippedResponse{},
	}

	have := make(map[string]bool, len(languages))
	for _, language := range languages {
		have[language.Code] = true
		out.Languages = append(out.Languages, newLanguageResponse(language, text[language.ID], shipped))
	}

	for _, file := range files {
		if !have[file.Code] {
			out.Shipped = append(out.Shipped, shippedResponse{Code: file.Code, Name: file.Name, NativeName: file.Native})
		}
	}

	return out
}
