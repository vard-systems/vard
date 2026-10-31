package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vard/migrations"
)

func TestANewChangelogIsTheNextNumber(t *testing.T) {
	all := migrations.All()
	next := all[len(all)-1].Version + 1

	tests := []struct {
		name     string
		input    string
		wantFile string
		wantErr  bool
	}{
		{"a name in lower case", "add_widgets", fmt.Sprintf("%05d_add_widgets.go", next), false},
		{"a name with spaces", "add widgets", "", true},
		{"a name in capitals", "AddWidgets", "", true},
		{"a path", "../add_widgets", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path, err := newChangelog(dir, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("newChangelog(%q) wrote %s, want it refused", tt.input, path)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Base(path) != tt.wantFile {
				t.Errorf("wrote %s, want %s", filepath.Base(path), tt.wantFile)
			}

			source, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := parser.ParseFile(token.NewFileSet(), path, source, 0); err != nil {
				t.Errorf("the changelog is not Go: %v", err)
			}
			if want := fmt.Sprintf("add(%d, upAddWidgets, downAddWidgets)", next); !strings.Contains(string(source), want) {
				t.Errorf("the changelog does not register itself as %s:\n%s", want, source)
			}

			if _, err := newChangelog(dir, tt.input); err == nil {
				t.Error("a second changelog of the same name was written over the first")
			}
		})
	}
}
