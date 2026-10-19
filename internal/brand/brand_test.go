package brand

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The name is written in three places and nowhere else: the constants in this
// package, the module path on the first line of go.mod, and one brand.ts in
// each app. These two tests are what make that true rather than merely
// intended — the first checks the three copies still agree, and the second
// fails on a fourth.

// The names a browser and the server have to agree on are written in all three
// places, so a rename that misses one is caught here rather than by a cookie
// that quietly stops being sent.
func TestTheAppsSpellTheSameNames(t *testing.T) {
	root := repoRoot(t)

	// What each app has to carry, beside the name and the slug every app has:
	// the console knows the admin session and where the project lives, and the
	// sign-in pages know the two the provider reads back.
	want := map[string][]string{
		"web/console/src/lib/brand.ts": {AdminSessionCookie, DocsURL, GitHubURL},
		"web/id/src/lib/brand.ts":      {UserSessionCookie, LanguageCookie},
	}

	for file, names := range want {
		t.Run(file, func(t *testing.T) {
			source := read(t, filepath.Join(root, filepath.FromSlash(file)))

			// The apps build their names from the slug and the name the way
			// this package does, so resolving the template literals first
			// compares what a browser would actually hold against what the
			// server would look for.
			resolved := strings.NewReplacer(
				"${BRAND.slug}", Slug,
				"${BRAND.name}", Name,
			).Replace(source)

			for _, expected := range append([]string{Name, Slug}, names...) {
				// The name and the addresses are written as they are; the names
				// built from the slug are template literals, so they arrive
				// here already resolved and quoted with backticks.
				if !strings.Contains(resolved, "'"+expected+"'") &&
					!strings.Contains(resolved, "`"+expected+"`") {
					t.Errorf("%s does not spell %q, which this package spells %q",
						file, expected, expected)
				}
			}
		})
	}
}

// Nothing outside the brand files may spell the name. A wordmark in a page
// title, a cookie in a handler, a product name in a comment, a fixture in a
// test: each of them is a copy that a rename would leave behind, and each of
// them is invisible until somebody looks for it.
func TestNothingElseSpellsTheName(t *testing.T) {
	root := repoRoot(t)

	// The module's own path opens an import in every file in the project, which
	// is a fact about the module rather than about the brand, so it is taken
	// out before looking at what is left.
	isImport := regexp.MustCompile(`"` + Slug + `/`)

	// The two spellings of the name: the slug an identifier is built from, and
	// the name a person reads. The environment prefix is deliberately not one
	// of them — brand.EnvPrefix is its own constant, config reads every setting
	// through it, and a comment may name the variable it is talking about.
	spellings := []string{Slug, Name}

	// The files allowed to spell the name: this package, and the one brand
	// module in each app.
	allowed := map[string]bool{
		filepath.FromSlash("web/console/src/lib/brand.ts"): true,
		filepath.FromSlash("web/id/src/lib/brand.ts"):      true,
	}

	skipped := map[string]bool{
		".git": true, ".logs": true, ".svelte-kit": true,
		"bin": true, "build": true, "node_modules": true,
	}

	// A command is named by its directory, so the project's own command spells
	// the name in cmd/<name>/ and has to: that is where `go run ./cmd/…` and
	// the build recipes find it. Renaming it is a `git mv` of the directory
	// beside the three edits, which is why the allowance is for whatever is
	// directly under cmd/ rather than for one name.
	command := filepath.FromSlash("cmd") + string(filepath.Separator)

	scanned := 0

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := entry.Name()
		if entry.IsDir() {
			if path != root && skipped[name] {
				return filepath.SkipDir
			}
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// brand.go and brand_test.go, which this directory is.
		if strings.HasPrefix(relative, filepath.FromSlash("internal"+string(filepath.Separator)+"brand")) {
			return nil
		}
		if allowed[relative] {
			return nil
		}
		if rest, under := strings.CutPrefix(relative, command); under &&
			strings.Count(rest, string(filepath.Separator)) == 1 {
			return nil
		}

		switch filepath.Ext(name) {
		case ".go", ".ts", ".svelte":
		default:
			return nil
		}

		scanned++

		source := read(t, path)
		source = isImport.ReplaceAllString(source, `""`)

		for _, spelling := range spellings {
			if strings.Contains(source, spelling) {
				t.Errorf("%s spells %q; the name belongs in internal/brand and the apps' brand.ts",
					filepath.ToSlash(relative), spelling)
			}
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// A walk that found nothing is a walk that stopped early, and a test that
	// passes because it looked at no files is worse than no test.
	if scanned < 200 {
		t.Errorf("only %d files were checked for the name; the walk did not reach the whole project", scanned)
	}
}

// repoRoot is the module's directory, which is where the two brand.ts files
// and every file the naming rule is about live. This package is two
// directories down from it, but walking up is what keeps that from mattering.
func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod is nowhere above this package, so the apps cannot be found")
		}
		dir = parent
	}
}

func read(t *testing.T, path string) string {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}

	return string(source)
}
