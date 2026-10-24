package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

// A language, end to end: English alone on a fresh installation, another added
// and translated in the panel, offered to the sign-in pages and served to them
// with its gaps filled in, made the default, and removed again.

type languageBody struct {
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Native    string         `json:"native_name"`
	Coverage  map[string]int `json:"coverage"`
	Missing   map[string]int `json:"missing"`
	Enabled   bool           `json:"is_enabled"`
	IsDefault bool           `json:"is_default"`
}

type languageList struct {
	Languages []languageBody `json:"languages"`
}

func (l languageList) find(code string) *languageBody {
	for i := range l.Languages {
		if l.Languages[i].Code == code {
			return &l.Languages[i]
		}
	}

	return nil
}

// public asks the public server, as the sign-in pages do.
func (s *liveServer) public(path string, out any) int {
	s.t.Helper()

	res, err := http.Get(s.root + "/api/v1/account" + path)
	if err != nil {
		s.t.Fatal(err)
	}
	defer res.Body.Close()

	if out != nil && res.StatusCode == http.StatusOK {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			s.t.Fatal(err)
		}
	}

	return res.StatusCode
}

func TestLiveLanguages(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	// A fresh installation has English and nothing else: whole, offered and
	// the default. Every other language is added here.
	var list languageList
	super.must(http.StatusOK, http.MethodGet, "/languages", nil, &list)

	english := list.find("en")
	switch {
	case len(list.Languages) != 1 || english == nil:
		t.Fatalf("languages = %+v, want English alone", list.Languages)
	case english.Coverage["id"] != 100:
		t.Errorf("English covers %d%% of the sign-in pages, want all of them", english.Coverage["id"])
	case !english.Enabled || !english.IsDefault:
		t.Errorf("English = %+v, want it offered and the default", english)
	}

	// The panel is written in English, in its own markup, so no language is
	// translated for it and none is counted against it.
	for _, language := range list.Languages {
		if _, counted := language.Coverage["console"]; counted {
			t.Errorf("%s is counted against the panel: %+v", language.Code, language.Coverage)
		}
	}

	// A language added from nothing and translated.
	var created struct{ Language languageBody }
	super.must(http.StatusCreated, http.MethodPost, "/languages", map[string]any{
		"code": "de", "name": "German", "native_name": "Deutsch",
	}, &created)

	if created.Language.Coverage["id"] != 0 {
		t.Errorf("a new language covers %d%% of the sign-in pages, want nothing yet", created.Language.Coverage["id"])
	}

	// The panel is shown in English only: a language an installation adds is a
	// sign-in language and nothing more.
	if _, counted := created.Language.Coverage["console"]; counted {
		t.Error("de was counted against the panel, which is not shown in it")
	}
	super.must(http.StatusNotFound, http.MethodGet, "/languages/de/translations/console", nil, nil)
	super.must(http.StatusNotFound, http.MethodPut, "/languages/de/translations/console", map[string]any{
		"messages": map[string]string{"nav.languages": "Sprachen"},
	}, nil)

	if status := s.public("/languages/de", nil); status != http.StatusNotFound {
		t.Errorf("a language that is off = %d to the sign-in pages, want 404", status)
	}

	super.must(http.StatusConflict, http.MethodPost, "/languages", map[string]any{
		"code": "DE", "name": "German", "native_name": "Deutsch",
	}, nil)

	var saved struct {
		Language languageBody `json:"language"`
		Ignored  int          `json:"ignored"`
	}
	super.must(http.StatusOK, http.MethodPut, "/languages/de/translations/id", map[string]any{
		"messages": map[string]string{"action.sign_in": "Anmelden", "nothing.here": "x", "$name": "Deutsch"},
	}, &saved)

	if saved.Ignored != 1 {
		t.Errorf("ignored = %d, want the one key nothing looks up", saved.Ignored)
	}
	if saved.Language.Missing["id"] != created.Language.Missing["id"]-1 {
		t.Errorf("missing = %d after translating one of %d, want one fewer",
			saved.Language.Missing["id"], created.Language.Missing["id"])
	}

	super.must(http.StatusNotFound, http.MethodPut, "/languages/de/translations/mobile", map[string]any{
		"messages": map[string]string{},
	}, nil)

	// Offered, the sign-in pages list it and get every key: its own where it
	// has one, English where it does not.
	super.must(http.StatusOK, http.MethodPatch, "/languages/de", map[string]any{"is_enabled": true}, nil)

	var offered struct {
		Languages []struct{ Code string } `json:"languages"`
		Default   string                  `json:"default"`
	}
	s.public("/languages", &offered)

	if len(offered.Languages) != 2 || offered.Languages[1].Code != "de" || offered.Default != "en" {
		t.Errorf("offered = %+v, want en then de, en the default", offered)
	}

	var text struct {
		Messages map[string]string `json:"messages"`
	}
	if status := s.public("/languages/de", &text); status != http.StatusOK {
		t.Fatalf("the text of an offered language = %d", status)
	}
	if text.Messages["action.sign_in"] != "Anmelden" {
		t.Errorf("action.sign_in = %q, want the German", text.Messages["action.sign_in"])
	}
	if text.Messages["login.title"] == "" {
		t.Error("a key German does not have came back empty rather than in English")
	}

	// Read once more — from the cache, when there is one — then reword it: the
	// sign-in pages have the new text on the very next read, not an hour on.
	s.public("/languages/de", &text)
	super.must(http.StatusOK, http.MethodPut, "/languages/de/translations/id", map[string]any{
		"messages": map[string]string{"action.sign_in": "Einloggen"},
	}, nil)
	s.public("/languages/de", &text)
	if text.Messages["action.sign_in"] != "Einloggen" {
		t.Errorf("after rewording, action.sign_in = %q, want the new text", text.Messages["action.sign_in"])
	}

	// English is what every other language falls back to, so rewording it
	// reaches German's untranslated keys at once too.
	super.must(http.StatusOK, http.MethodPut, "/languages/en/translations/id", map[string]any{
		"messages": map[string]string{"login.title": "Welcome back"},
	}, nil)
	s.public("/languages/de", &text)
	if text.Messages["login.title"] != "Welcome back" {
		t.Errorf("German's fallback for login.title = %q, want the reworded English", text.Messages["login.title"])
	}

	// The two languages nothing may remove: the base, and the default.
	super.must(http.StatusBadRequest, http.MethodDelete, "/languages/en", nil, nil)
	super.must(http.StatusOK, http.MethodPatch, "/languages/de", map[string]any{"is_default": true}, nil)
	super.must(http.StatusBadRequest, http.MethodDelete, "/languages/de", nil, nil)
	super.must(http.StatusBadRequest, http.MethodPatch, "/languages/de", map[string]any{"is_enabled": false, "is_default": false}, nil)

	super.must(http.StatusOK, http.MethodGet, "/languages", nil, &list)
	if list.find("en").IsDefault || !list.find("de").IsDefault {
		t.Errorf("making de the default left en = %+v, de = %+v", list.find("en"), list.find("de"))
	}

	super.must(http.StatusOK, http.MethodPatch, "/languages/en", map[string]any{"is_default": true}, nil)
	super.must(http.StatusNoContent, http.MethodDelete, "/languages/de", nil, nil)

	// A language copied from another starts with all of its text, and one
	// added as the default is offered at once and takes the mark.
	super.must(http.StatusCreated, http.MethodPost, "/languages", map[string]any{
		"code": "en-GB", "name": "English (UK)", "native_name": "English (UK)",
		"copy_from": "en", "is_default": true,
	}, &created)

	if created.Language.Coverage["id"] != 100 || !created.Language.Enabled || !created.Language.IsDefault {
		t.Errorf("en-GB = %+v, want all of English's text, offered and the default", created.Language)
	}
	super.must(http.StatusOK, http.MethodGet, "/languages", nil, &list)
	if list.find("en").IsDefault {
		t.Error("adding a default language left English the default as well")
	}
	s.public("/languages", &offered)
	if offered.Default != "en-GB" {
		t.Errorf("the sign-in pages' default = %q, want en-GB", offered.Default)
	}

	super.must(http.StatusBadRequest, http.MethodPost, "/languages", map[string]any{
		"code": "fr", "name": "French", "native_name": "Français", "copy_from": "it",
	}, nil)

	// The sign-in pages are the only app a language is translated for, so the
	// panel is neither counted nor writable through this.
	if _, counted := created.Language.Coverage["console"]; counted {
		t.Errorf("the panel is counted as translated: %+v", created.Language.Coverage)
	}
	super.must(http.StatusNotFound, http.MethodPut, "/languages/en-GB/translations/console", map[string]any{
		"messages": map[string]string{"error.unknown": "…"},
	}, nil)
}
