package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The code is in layers, each allowed what is below it and nothing above:
// the domain knows no database, the infrastructure no use case, and only the
// interfaces answer requests. brand, config and jose are shared by all.
var layers = map[string][]string{
	"brand":          {},
	"config":         {"brand"},
	"jose":           {},
	"domain":         {"brand", "config", "jose", "domain"},
	"infrastructure": {"brand", "config", "jose", "domain", "infrastructure"},
	"application":    {"brand", "config", "jose", "domain", "infrastructure", "application"},
	"interfaces":     {"brand", "config", "jose", "domain", "infrastructure", "application", "interfaces"},
}

// The top-level packages beside internal/ are for the layers that need them.
var beside = map[string][]string{
	"migrations": {"infrastructure"},
	"i18n":       {"infrastructure", "application", "interfaces"},
}

const module = "vard/"

func TestEachLayerImportsOnlyWhatIsBelowIt(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		layer, _, _ := strings.Cut(filepath.ToSlash(path), "/")
		allowed, known := layers[layer]
		if !known {
			if !strings.Contains(path, string(filepath.Separator)) {
				return nil
			}
			t.Errorf("%s is in no layer; put it in one of %v", path, keys())
			return nil
		}

		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, spec := range file.Imports {
			imported, _ := strconv.Unquote(spec.Path.Value)
			if !strings.HasPrefix(imported, module) {
				continue
			}
			inside := strings.TrimPrefix(imported, module)
			if top, ok := strings.CutPrefix(inside, "internal/"); ok {
				to, _, _ := strings.Cut(top, "/")
				if !slices.Contains(allowed, to) {
					t.Errorf("%s (%s) imports %s (%s), which is above it", path, layer, imported, to)
				}
				continue
			}
			top, _, _ := strings.Cut(inside, "/")
			if !slices.Contains(beside[top], layer) {
				t.Errorf("%s (%s) imports %s, which is for %v", path, layer, imported, beside[top])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func keys() []string {
	out := make([]string, 0, len(layers))
	for layer := range layers {
		out = append(out, layer)
	}
	return out
}
