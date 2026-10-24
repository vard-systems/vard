package i18n

import (
	"encoding/json"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// English is the only language that ships: every other is added in the panel.
func TestOnlyEnglishShips(t *testing.T) {
	err := fs.WalkDir(files, ".", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		if !strings.HasSuffix(name, "."+Base+".json") {
			t.Errorf("%s ships: only %s files belong in i18n/, other languages are added in the panel", name, Base)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Every key in the files is read: go-i18n takes a group whose keys are all
// words such as "other" or "description" for a plural message, which would
// lose them silently.
func TestEveryKeyIsRead(t *testing.T) {
	for _, app := range catalogued {
		t.Run(string(app), func(t *testing.T) {
			paths, err := fs.Glob(files, string(app)+"/*.json")
			if err != nil {
				t.Fatal(err)
			}

			want := 0
			for _, path := range paths {
				want += len(leaves(t, path))
			}

			if got := len(Keys(app)); got != want || got == 0 {
				t.Errorf("%d keys read, the files have %d", got, want)
			}
		})
	}
}

// The groups are organised the one way: semantic groups nested by namespace,
// and every part of a key lower case with underscores.
func TestKeysAreNestedAndLowerCase(t *testing.T) {
	part := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

	for _, app := range catalogued {
		paths, _ := fs.Glob(files, string(app)+"/*.json")

		for _, path := range paths {
			for _, key := range leaves(t, path) {
				for piece := range strings.SplitSeq(key, ".") {
					if !part.MatchString(piece) {
						t.Errorf("%s: %q is not nested lower case with underscores", path, key)
						break
					}
				}
			}
		}
	}
}

// The admin API's English is the console's errors and nothing else, and it is
// never offered for translation.
func TestConsoleTextIsTheAdminAPIsEnglish(t *testing.T) {
	keys := Keys(Console)
	if len(keys) == 0 {
		t.Fatal("i18n/console has no keys: the admin API has no English to answer with")
	}

	for _, key := range keys {
		if !strings.HasPrefix(key, "error.") {
			t.Errorf("console holds %q: it is the admin API's error sentences and nothing else", key)
		}
	}

	if slices.Contains(Apps, Console) {
		t.Error("the panel is offered for translation: it is written in English")
	}
}

func TestCatalog(t *testing.T) {
	kyrgyz := NewCatalog("ky", Messages{"login.title": "Кирүү", "login.hello": "Салам, {name}!"})

	tests := []struct {
		name    string
		catalog *Catalog
		key     string
		params  map[string]any
		want    string
	}{
		{name: "a key in the language", catalog: kyrgyz, key: "login.title", want: "Кирүү"},
		{name: "a parameter filled in", catalog: kyrgyz, key: "login.hello", params: map[string]any{"name": "Ада"}, want: "Салам, Ада!"},
		{name: "a parameter nobody passed", catalog: kyrgyz, key: "login.hello", want: "Салам, {name}!"},
		{name: "a key the language lacks", catalog: kyrgyz, key: "nothing.here", want: "nothing.here"},
		{name: "a code go-i18n cannot read", catalog: NewCatalog("not a tag", Messages{"a": "{n} left"}), key: "a", params: map[string]any{"n": 3}, want: "3 left"},
		{name: "the shipped English", catalog: English(ID), key: "login.subtitle_app", params: map[string]any{"app": "Shop"}, want: "to continue to Shop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.catalog.Localize(tt.key, tt.params); got != tt.want {
				t.Errorf("Localize(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	if !kyrgyz.Has("login.title") || kyrgyz.Has("nothing.here") {
		t.Error("Has does not tell a key the catalog has from one it lacks")
	}
}

func TestCoverage(t *testing.T) {
	keys := Keys(ID)

	tests := []struct {
		name        string
		messages    Messages
		wantMissing int
	}{
		{name: "nothing translated", messages: Messages{}, wantMissing: len(keys)},
		{name: "one key", messages: Messages{keys[0]: "x"}, wantMissing: len(keys) - 1},
		{name: "a key that is only spaces", messages: Messages{keys[0]: "  "}, wantMissing: len(keys)},
		{name: "a key nothing looks up", messages: Messages{"nothing.here": "x"}, wantMissing: len(keys)},
		{name: "the shipped English", messages: Resolve(ID), wantMissing: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, missing := Coverage(ID, tt.messages); missing != tt.wantMissing {
				t.Errorf("missing = %d, want %d", missing, tt.wantMissing)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	const key = "action.sign_in"

	english, _ := Text(ID, key)
	if english == "" {
		t.Fatalf("the shipped English has no %s", key)
	}

	tests := []struct {
		name   string
		layers []Messages
		want   string
	}{
		{name: "the language has the key", layers: []Messages{{key: "Кирүү"}, {key: "Log in"}}, want: "Кирүү"},
		{name: "the language is missing it", layers: []Messages{{}, {key: "Log in"}}, want: "Log in"},
		{name: "an empty translation is missing", layers: []Messages{{key: " "}, {key: "Log in"}}, want: "Log in"},
		{name: "nobody has it", layers: []Messages{{}, {}}, want: english},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Resolve(ID, tt.layers...)[key]; got != tt.want {
				t.Errorf("Resolve()[%q] = %q, want %q", key, got, tt.want)
			}
		})
	}

	if _, kept := Resolve(ID, Messages{"nothing.here": "x"})["nothing.here"]; kept {
		t.Error("Resolve kept a key the shipped English does not have")
	}
}

func TestKnown(t *testing.T) {
	keys := Keys(ID)

	kept, dropped := Known(ID, Messages{"$name": "German", keys[0]: "Hallo", keys[1]: "  ", "nothing.here": "x"})

	if !maps.Equal(kept, Messages{keys[0]: "Hallo"}) {
		t.Errorf("kept = %v, want only %s", kept, keys[0])
	}
	if dropped != 1 {
		t.Errorf("dropped = %d, want 1: the unknown key, not the name or the empty one", dropped)
	}
}

func TestParseApp(t *testing.T) {
	tests := []struct {
		name string
		app  string
		ok   bool
	}{
		{name: "the sign-in pages", app: "id", ok: true},
		{name: "the panel, which is not translated", app: "console"},
		{name: "a path", app: "../id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := ParseApp(tt.app); ok != tt.ok {
				t.Errorf("ParseApp(%q) ok = %v, want %v", tt.app, ok, tt.ok)
			}
		})
	}
}

// leaves are a file's keys, flattened the plain way, to hold go-i18n's
// reading to.
func leaves(t *testing.T, path string) []string {
	t.Helper()

	raw, err := files.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var top map[string]any
	if err := json.Unmarshal(raw, &top); err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	var out []string
	var walk func(prefix string, node map[string]any)
	walk = func(prefix string, node map[string]any) {
		for key, value := range node {
			if prefix != "" {
				key = prefix + "." + key
			}
			switch value := value.(type) {
			case map[string]any:
				walk(key, value)
			case string:
				out = append(out, key)
			default:
				t.Errorf("%s: %s is not text", path, key)
			}
		}
	}
	walk("", top)

	return out
}
