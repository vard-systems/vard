package languages

import (
	"time"

	"github.com/google/uuid"

	"vard/i18n"
	"vard/internal/domain/model"
)

// languageResponse is one language as the panel sees it: its settings, and
// how much of each app it translates.
type languageResponse struct {
	Code string `json:"code"`
	// Name is what the language is called in English, Native what it calls
	// itself.
	Name       string `json:"name"`
	NativeName string `json:"native_name"`

	// Apps are the apps a language is translated for: the sign-in pages.
	Apps []i18n.App `json:"apps"`

	// Coverage and Missing are per app: the percentage of base keys translated
	// and how many are missing.
	Coverage map[i18n.App]int `json:"coverage"`
	Missing  map[i18n.App]int `json:"missing"`

	IsEnabled bool `json:"is_enabled"`
	IsDefault bool `json:"is_default"`
	Position  int  `json:"position"`

	// Base marks English, which ships with the server and which every other
	// language falls back to: it cannot be turned off or removed, and what is
	// stored for it only changes the shipped text.
	Base bool `json:"base"`

	UpdatedAt time.Time `json:"updated_at"`
}

// listResponse is what the list endpoint answers with.
type listResponse struct {
	Languages []languageResponse `json:"languages"`
	// Apps are the two catalogs a language is counted against, so the panel
	// names the columns from the server rather than from a list of its own.
	Apps []i18n.App `json:"apps"`
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

func newLanguageResponse(language model.Language, text map[string]map[string]string) languageResponse {
	base := language.Code == model.BaseLanguage

	out := languageResponse{
		Code:       language.Code,
		Name:       language.Name,
		NativeName: language.NativeName,
		Apps:       i18n.Apps,
		Coverage:   map[i18n.App]int{},
		Missing:    map[i18n.App]int{},
		IsEnabled:  language.IsEnabled,
		IsDefault:  language.IsDefault,
		Position:   language.Position,
		Base:       base,
		UpdatedAt:  language.UpdatedAt,
	}

	for _, app := range out.Apps {
		messages := text[string(app)]
		if base {
			messages = i18n.Resolve(app, messages)
		}
		out.Coverage[app], out.Missing[app] = i18n.Coverage(app, messages)
	}

	return out
}

func newListResponse(languages []model.Language, text map[uuid.UUID]map[string]map[string]string) listResponse {
	out := listResponse{
		Languages: make([]languageResponse, 0, len(languages)),
		Apps:      i18n.Apps,
	}

	for _, language := range languages {
		out.Languages = append(out.Languages, newLanguageResponse(language, text[language.ID]))
	}

	return out
}
