package brand

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Each application keeps its own brand: this package for the server, and
// src/lib/brand.ts in each app. They may call the product, draw its mark and
// pick their favicon as they like, but they build cookie names from one slug,
// and the server reads back what the apps' cookies are called. These tests keep
// those agreeing, and the name out of everywhere else.

// brands are the apps' brand files, and the cookies each one names that the
// server reads back: the console's admin session, the sign-in pages' user
// session (under both names, plain http's and https's __Host- one) and
// language.
var brands = map[string][]string{
	"web/console/src/lib/brand.ts": {AdminSessionCookie, HostPrefix + AdminSessionCookie},
	"web/id/src/lib/brand.ts":      {UserSessionCookie, HostPrefix + UserSessionCookie, LanguageCookie},
}

// The slug becomes cookie, Redis, URN, Postgres and environment names.
func TestTheSlugIsAnIdentifier(t *testing.T) {
	if !regexp.MustCompile(`^[a-z][a-z0-9]*$`).MatchString(Slug) {
		t.Errorf("Slug = %q, want lower-case letters and digits, starting with a letter", Slug)
	}
	if EnvPrefix != strings.ToUpper(Slug)+"_" {
		t.Errorf("EnvPrefix = %q, want the slug in capitals and an underscore", EnvPrefix)
	}
}

// A change to one side that misses the other is caught here rather than by a
// cookie that quietly stops being sent.
func TestTheAppsBuildTheSameNames(t *testing.T) {
	root := repoRoot(t)

	for file, cookies := range brands {
		t.Run(file, func(t *testing.T) {
			source := read(t, filepath.Join(root, filepath.FromSlash(file)))

			if !strings.Contains(source, "slug: '"+Slug+"'") {
				t.Errorf("%s has no slug: '%s', which the server has", file, Slug)
			}

			// Resolving the template literals first compares what a browser
			// would hold against what the server would look for.
			resolved := strings.ReplaceAll(source, "${BRAND.slug}", Slug)
			for _, cookie := range cookies {
				if !strings.Contains(resolved, "`"+cookie+"`") {
					t.Errorf("%s does not build %q, which the server reads back", file, cookie)
				}
			}
		})
	}
}

// Nothing else may spell the name. A wordmark in a page
// title, a cookie in a handler, a product name in a comment, a fixture in a
// test: each of them is a copy that a rename would leave behind, and each of
// them is invisible until somebody looks for it.
func TestNothingElseSpellsTheName(t *testing.T) {
	root := repoRoot(t)

	// The module's own path opens an import in every file in the project, which
	// is a fact about the module rather than about the brand, so it is taken
	// out before looking at what is left.
	isImport := regexp.MustCompile(`"` + Slug + `/`)

	// Go asks a command's doc comment to open with its name, which is its
	// directory's: cmd/<slug>/.
	isCommandDoc := regexp.MustCompile(`(?m)^// Command ` + Slug + `\b`)

	// The two spellings of the name: the slug an identifier is built from, and
	// the name a person reads. The environment prefix is deliberately not one
	// of them — brand.EnvPrefix is its own constant, config reads every setting
	// through it, and a comment may name the variable it is talking about.
	spellings := []string{Slug, Name}

	skipped := map[string]bool{
		".git": true, ".logs": true, ".svelte-kit": true,
		"bin": true, "build": true, "node_modules": true,
	}

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

		// The brands themselves: this package, and each app's brand file.
		if strings.HasPrefix(relative, filepath.FromSlash("internal/brand/")) {
			return nil
		}
		if _, isBrand := brands[filepath.ToSlash(relative)]; isBrand {
			return nil
		}

		// What the code is made of. The scripts, deploy/, CI and the docs
		// spell the name too, where a tool needs it written out: scripts/rename.sh
		// rewrites those.
		switch filepath.Ext(name) {
		case ".go", ".ts", ".svelte", ".js", ".json", ".html", ".css", ".svg":
		default:
			return nil
		}

		scanned++

		source := read(t, path)
		source = isImport.ReplaceAllString(source, `""`)
		source = isCommandDoc.ReplaceAllString(source, "")

		for _, spelling := range spellings {
			if strings.Contains(source, spelling) {
				t.Errorf("%s spells %q; the name belongs in the brands: internal/brand and each app's src/lib/brand.ts",
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

// repoRoot is the module's directory, where the apps and every file the
// naming rule is about live.
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
