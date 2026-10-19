package apidoc

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// The server answers these itself (internal/api/reference), so they are
// committed beside the handler that embeds them.
const serverDocuments = "internal/api/reference"

func Generate(root string) (map[string][]byte, error) {
	b, err := build(root)
	if err != nil {
		return nil, err
	}

	files := map[string][]byte{}
	for _, server := range b.Servers {
		document, err := b.openAPI(server)
		if err != nil {
			return nil, err
		}
		files[serverDocuments+"/"+server.Key+".json"] = document
	}

	return files, nil
}

func Stale(root string, files map[string][]byte) ([]string, error) {
	var out []string
	for path, want := range files {
		have, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil || !bytes.Equal(have, want) {
			out = append(out, path)
		}
	}

	sort.Strings(out)
	return out, nil
}

func Write(root string, files map[string][]byte) error {
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			return fmt.Errorf("apidoc: writing %s: %w", path, err)
		}
	}

	return nil
}
