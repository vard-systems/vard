package api

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Every action the server writes to the activity log has a sentence in the
// panel (web/console/src/lib/components/activity/actions.ts); one without
// shows its raw name, "social_provider.created something".
func TestEveryRecordedActionHasASentence(t *testing.T) {
	known := map[string]bool{}
	for _, match := range regexp.MustCompile(`(?m)^\t'([a-z_]+\.[a-z_]+)': \{`).FindAllStringSubmatch(read(t, "../web/console/src/lib/components/activity/actions.ts"), -1) {
		known[match[1]] = true
	}
	if len(known) == 0 {
		t.Fatal("no actions found in actions.ts")
	}

	// The resources the log speaks of; a literal like "users.email" is a
	// column, and one ending in .read or .write a permission, not an action.
	action := regexp.MustCompile(`"((?:admin|admin_role|admin_security|api|application|cache|language|login_flow|mail|organization|otp|signing_keys|social_provider|sso_connection|user|user_field|user_role)\.[a-z_]+)"`)

	var missing []string
	// Every layer records: the handlers here and the use cases in core/
	// alike, so the whole project is read.
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		for _, match := range action.FindAllStringSubmatch(read(t, path), -1) {
			permission := strings.HasSuffix(match[1], ".read") || strings.HasSuffix(match[1], ".write")
			if !permission && !known[match[1]] && !slices.Contains(missing, match[1]) {
				missing = append(missing, match[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	slices.Sort(missing)
	for _, name := range missing {
		t.Errorf("%s is recorded but has no sentence in actions.ts", name)
	}
}
