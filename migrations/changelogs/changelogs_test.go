package changelogs

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// A file's number is the version it registers, so the folder reads as the
// order goose applies them in.
func TestEachFileRegistersItsOwnNumber(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") && name != "changelogs.go" {
			files = append(files, name)
		}
	}

	registered := All()
	if len(registered) != len(files) {
		t.Fatalf("%d files register %d changelogs; each file adds exactly one", len(files), len(registered))
	}
	for i, file := range files {
		if want := fmt.Sprintf("%05d_", registered[i].Version); !strings.HasPrefix(file, want) {
			t.Errorf("%s registers version %d; name it %s…", file, registered[i].Version, want)
		}
	}
}
