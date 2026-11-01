package layers_test

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

// The Go code is in three folders, PocketBase's way: tools/ (building blocks
// that know nothing of this server), core/ (the server itself) and api/ (HTTP).
// Inside them each package is in a layer, allowed what is below it and nothing
// above: the model knows no database, the infrastructure no use case, and only
// api answers requests.
var layerOf = map[string]string{
	"tools":         "tools",
	"core/brand":    "brand",
	"core/config":   "config",
	"core/model":    "model",
	"core/cache":    "infrastructure",
	"core/database": "infrastructure",
	"core/mail":     "infrastructure",
	"core/passkey":  "infrastructure",
	"core/store":    "infrastructure",
	"core/auth":     "service",
	"core/oidc":     "service",
	"api":           "api",
}

var layers = map[string][]string{
	"tools":          {"tools"},
	"brand":          {},
	"config":         {"brand"},
	"model":          {"tools", "brand", "config", "model"},
	"infrastructure": {"tools", "brand", "config", "model", "infrastructure"},
	"service":        {"tools", "brand", "config", "model", "infrastructure", "service"},
	"api":            {"tools", "brand", "config", "model", "infrastructure", "service", "api"},
}

// The other top-level packages are for the layers that need them.
var beside = map[string][]string{
	"migrations": {"infrastructure"},
	"i18n":       {"infrastructure", "service", "api"},
}

var walked = []string{"tools", "core", "api"}

const module = "vard/"

func TestEachLayerImportsOnlyWhatIsBelowIt(t *testing.T) {
	for _, root := range walked {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			path = filepath.ToSlash(path)
			layer, known := layerFor(path)
			if !known {
				t.Errorf("%s is in no layer; add its folder to layerOf", path)
				return nil
			}
			allowed := layers[layer]

			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, spec := range file.Imports {
				imported, _ := strconv.Unquote(spec.Path.Value)
				inside, ok := strings.CutPrefix(imported, module)
				if !ok {
					continue
				}
				if to, known := layerFor(inside + "/"); known {
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
}

// layerFor is the layer of the longest folder in layerOf that holds path.
func layerFor(path string) (string, bool) {
	best, layer := "", ""
	for dir, l := range layerOf {
		if strings.HasPrefix(path, dir+"/") && len(dir) > len(best) {
			best, layer = dir, l
		}
	}
	return layer, best != ""
}
