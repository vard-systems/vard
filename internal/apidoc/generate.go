package apidoc

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const (
	serverDocuments = "internal/api/reference"
	docsDocuments   = "web/docs/static/openapi"
	docsCollections = "web/docs/static/collections"
	docsPages       = "web/docs/src/content/reference"
)

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
		files[docsDocuments+"/"+server.Key+".json"] = document

		collection, err := b.collection(server)
		if err != nil {
			return nil, err
		}
		files[docsCollections+"/"+server.Key+".postman_collection.json"] = collection
	}

	for _, p := range b.markdown() {
		files[docsPages+"/"+p.path] = p.bytes()
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

	orphans, err := orphans(root, files)
	if err != nil {
		return nil, err
	}
	out = append(out, orphans...)

	sort.Strings(out)
	return out, nil
}

func Write(root string, files map[string][]byte) error {
	orphans, err := orphans(root, files)
	if err != nil {
		return err
	}
	for _, path := range orphans {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			return err
		}
	}

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

func orphans(root string, files map[string][]byte) ([]string, error) {
	var out []string
	dir := filepath.Join(root, filepath.FromSlash(docsPages))

	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return filepath.SkipDir
		}
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if _, made := files[filepath.ToSlash(relative)]; !made {
			out = append(out, filepath.ToSlash(relative))
		}
		return nil
	})

	return out, err
}
