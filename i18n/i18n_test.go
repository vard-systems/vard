package i18n

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Every shipped language has to name itself, and the base language has to be
// there at all: it is what a missing translation falls back to, and what
// coverage is a percentage of.
func TestShipped(t *testing.T) {
	shipped, err := Shipped()
	if err != nil {
		t.Fatalf("Shipped() = %v", err)
	}

	if len(shipped) == 0 || shipped[0].Code != Base {
		t.Fatalf("the shipped languages start with %v, want the base language first", shipped)
	}

	for _, language := range shipped {
		t.Run(language.Code, func(t *testing.T) {
			switch {
			case language.Name == "" || language.Name == language.Code:
				t.Errorf("$name is missing: a language has to say what it is called")
			case language.Native == "":
				t.Error("$native is missing")
			}

			for app, messages := range language.Messages {
				if _, meta := messages["$name"]; meta {
					t.Errorf("%s still carries $name among its messages for %s", language.Code, app)
				}
			}
		})
	}
}

// A language that ships has to be complete for every app it is for: the
// first start imports it, and an installation that offers it should not be
// offering half a page.
func TestTranslationsAreComplete(t *testing.T) {
	shipped, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}

	for _, language := range shipped {
		for _, app := range AppsFor(language.Code) {
			if _, has := language.Messages[app]; !has {
				t.Errorf("%s has no groups for %s", language.Code, app)
				continue
			}

			if percent, missing := Coverage(app, language.Messages[app]); missing > 0 {
				t.Errorf("%s is %d%% of %s, short %d keys; add them to its groups",
					language.Code, percent, app, missing)
			}
		}

		for app := range language.Messages {
			if !ServesApp(language.Code, app) {
				t.Errorf("i18n/%s/%s/ ships, but %s is not an app a language is translated for", app, language.Code, app)
			}
		}
	}
}

// Every language is on the sign-in pages, and on nothing else: the panel is
// written in English, in its own markup, and is not translated.
func TestAppsFor(t *testing.T) {
	for _, code := range []string{"en", "ru", "de", "pt-BR"} {
		if got, want := AppsFor(code), []App{ID}; !slices.Equal(got, want) {
			t.Errorf("AppsFor(%q) = %v, want %v", code, got, want)
		}
		if ServesApp(code, Console) {
			t.Errorf("ServesApp(%q, Console) = true: the panel is not translated", code)
		}
	}
}

// The admin API's English is still read the same way, even though no language
// is translated for it: it is where a problem the panel is shown keeps its
// sentence.
func TestConsoleTextIsTheAdminAPIsEnglish(t *testing.T) {
	keys := Keys(Console)
	if len(keys) == 0 {
		t.Fatal("i18n/console/en/ has no keys: the admin API has no English to answer with")
	}

	for _, key := range keys {
		if !strings.HasPrefix(key, "error.") {
			t.Errorf("console holds %q: it is the admin API's error sentences and nothing else", key)
		}

		if text, ok := Text(Console, key); !ok || strings.TrimSpace(text) == "" {
			t.Errorf("%s has no English", key)
		}
	}

	// And it never becomes a language: nothing imports it into the database.
	shipped, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range shipped {
		if _, has := file.Messages[Console]; has {
			t.Errorf("%s ships panel text, which would be imported as a translation", file.Code)
		}
	}
}

func TestCoverage(t *testing.T) {
	keys := Keys(ID)
	if len(keys) < 2 {
		t.Fatal("the base language has too few keys to test coverage with")
	}

	tests := []struct {
		name        string
		messages    Messages
		wantMissing int
	}{
		{name: "nothing translated", messages: Messages{}, wantMissing: len(keys)},
		{name: "one key", messages: Messages{keys[0]: "x"}, wantMissing: len(keys) - 1},
		{name: "a key that is only spaces", messages: Messages{keys[0]: "  "}, wantMissing: len(keys)},
		{name: "a key nothing looks up", messages: Messages{"nothing.here": "x"}, wantMissing: len(keys)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, missing := Coverage(ID, tt.messages)
			if missing != tt.wantMissing {
				t.Errorf("missing = %d, want %d", missing, tt.wantMissing)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	const key = "action.sign_in"

	shippedEnglish := Resolve(ID)[key]
	if shippedEnglish == "" {
		t.Fatalf("the base language has no %s", key)
	}

	tests := []struct {
		name   string
		layers []Messages
		want   string
	}{
		{name: "the language has the key", layers: []Messages{{key: "Кирүү"}, {key: "Log in"}}, want: "Кирүү"},
		{name: "the language is missing it", layers: []Messages{{}, {key: "Log in"}}, want: "Log in"},
		{name: "an empty translation is missing", layers: []Messages{{key: " "}, {key: "Log in"}}, want: "Log in"},
		{name: "nobody has it", layers: []Messages{{}, {}}, want: shippedEnglish},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Resolve(ID, tt.layers...)[key]; got != tt.want {
				t.Errorf("Resolve()[%q] = %q, want %q", key, got, tt.want)
			}
		})
	}

	if _, kept := Resolve(ID, Messages{"nothing.here": "x"})["nothing.here"]; kept {
		t.Error("Resolve kept a key the base language does not have")
	}

	if got, want := len(Resolve(Console)), len(Keys(Console)); got != want {
		t.Errorf("Resolve(Console) has %d keys, want every one of the %d", got, want)
	}
}

func TestKnown(t *testing.T) {
	key := Keys(ID)[0]

	kept, dropped := Known(ID, Messages{
		"$name":        "Deutsch",
		key:            "Hallo",
		Keys(ID)[1]:    "  ",
		"nothing.here": "x",
	})

	if len(kept) != 1 || kept[key] != "Hallo" {
		t.Errorf("kept = %v, want only %s", kept, key)
	}
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1: the unknown key, not the name or the empty one", dropped)
	}
}

func TestParseApp(t *testing.T) {
	for _, app := range Apps {
		if got, ok := ParseApp(string(app)); !ok || got != app {
			t.Errorf("ParseApp(%q) = %q, %v", app, got, ok)
		}
	}

	if _, ok := ParseApp("../id"); ok {
		t.Error("ParseApp accepted something that is not an app")
	}
}

func TestFlatten(t *testing.T) {
	tests := []struct {
		name    string
		file    map[string]any
		want    Messages
		wantErr bool
	}{
		{
			name: "nested by namespace",
			file: map[string]any{"$name": "English", "login": map[string]any{"title": "Sign in", "form": map[string]any{"email": "Email"}}},
			want: Messages{"$name": "English", "login.title": "Sign in", "login.form.email": "Email"},
		},
		{
			name: "flat, as the database holds it",
			file: map[string]any{"login.title": "Sign in"},
			want: Messages{"login.title": "Sign in"},
		},
		{
			name:    "the same key said twice",
			file:    map[string]any{"login.title": "Sign in", "login": map[string]any{"title": "Log in"}},
			wantErr: true,
		},
		{
			name:    "a value that is not text",
			file:    map[string]any{"login": map[string]any{"attempts": 3.0}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Flatten(tt.file)
			switch {
			case tt.wantErr && err == nil:
				t.Fatalf("Flatten() = %v, want an error", got)
			case tt.wantErr:
				return
			case err != nil:
				t.Fatal(err)
			}

			if len(got) != len(tt.want) {
				t.Fatalf("Flatten() = %v, want %v", got, tt.want)
			}
			for key, value := range tt.want {
				if got[key] != value {
					t.Errorf("%s = %q, want %q", key, got[key], value)
				}
			}
		})
	}
}

// The shipped groups are organised the one way: each app has a directory per
// language, semantic JSON groups inside it, and every part of a key lower case
// with underscores. A group written flat, or a key spelled "loginTitle", is
// caught here rather than in review.
func TestShippedGroupsAreNested(t *testing.T) {
	part := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	for _, app := range catalogued {
		err := fs.WalkDir(files, string(app), func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(name, ".json") {
				return nil
			}

			raw, err := files.ReadFile(name)
			if err != nil {
				return err
			}

			var top map[string]any
			if err := json.Unmarshal(raw, &top); err != nil {
				t.Fatalf("%s: %v", name, err)
			}

			for key := range top {
				if strings.Contains(key, ".") {
					t.Errorf("%s has %q at the top: nest it under %q", name, key, strings.Split(key, ".")[0])
				}
			}

			flat, err := Flatten(top)
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			for key := range flat {
				if strings.HasPrefix(key, metaPrefix) {
					continue
				}
				for piece := range strings.SplitSeq(key, ".") {
					if !part.MatchString(piece) {
						t.Errorf("%s: %q is not lower case with underscores", name, key)
						break
					}
				}
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
