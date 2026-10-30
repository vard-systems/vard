// Package tstypes holds each API package's answers to the types the panel
// declares for them in web/console/src/lib/api/types.ts, which are written by
// hand: a field renamed on one side and not the other would otherwise only
// show as something missing on a page. Tests use it; nothing else does.
package tstypes

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// Panel is the panel's types file, from the repository's root.
const Panel = "web/console/src/lib/api/types.ts"

// Match fails the test when the TypeScript type `name` and the JSON that
// `value` encodes to do not have the same fields. `serverOnly` names fields
// the server sends that the panel deliberately leaves undeclared.
func Match(t *testing.T, name string, value any, serverOnly ...string) {
	t.Helper()

	declared, err := Fields(source(t), name)
	if err != nil {
		t.Fatal(err)
	}
	sent := JSONFields(reflect.TypeOf(value))

	for _, field := range sent {
		if !slices.Contains(declared, field) && !slices.Contains(serverOnly, field) {
			t.Errorf("%s: the server sends %q, which the panel's type does not declare", name, field)
		}
	}
	for _, field := range declared {
		if !slices.Contains(sent, field) {
			t.Errorf("%s: the panel's type declares %q, which the server never sends", name, field)
		}
	}
}

var cached string

func source(t *testing.T) string {
	t.Helper()

	if cached != "" {
		return cached
	}

	_, here, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(here), "..", "..", "..", "..", "..")

	data, err := os.ReadFile(filepath.Join(root, Panel))
	if err != nil {
		t.Fatal(err)
	}
	cached = string(data)

	return cached
}

// JSONFields are the names a struct encodes its fields under: embedded
// structs' fields in their place, "-" and unexported fields left out.
func JSONFields(typ reflect.Type) []string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	var names []string
	for i := range typ.NumField() {
		field := typ.Field(i)
		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")

		switch {
		case tag == "-":
		case field.Anonymous && tag == "":
			names = append(names, JSONFields(field.Type)...)
		case !field.IsExported():
		case tag != "":
			names = append(names, tag)
		default:
			names = append(names, field.Name)
		}
	}

	return names
}

var declaration = regexp.MustCompile(`(?m)^export type (\w+)(?:<[^=]*>)? = `)

// Fields are the top-level fields of an object type in a TypeScript source,
// with those of the types it is joined to with &.
func Fields(source, name string) ([]string, error) {
	start := -1
	for _, match := range declaration.FindAllStringSubmatchIndex(source, -1) {
		if source[match[2]:match[3]] == name {
			start = match[1]
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("types.ts declares no type %s", name)
	}

	rest := source[start:]
	open := strings.Index(rest, "{")
	if open < 0 {
		return nil, fmt.Errorf("type %s is not an object type", name)
	}

	var fields []string
	for joined := range strings.SplitSeq(rest[:open], "&") {
		joined = strings.TrimSpace(joined)
		if joined == "" {
			continue
		}
		inherited, err := Fields(source, joined)
		if err != nil {
			return nil, fmt.Errorf("type %s joins %s: %w", name, joined, err)
		}
		fields = append(fields, inherited...)
	}

	return append(fields, topLevel(rest[open:])...), nil
}

var fieldName = regexp.MustCompile(`^\s*(?:readonly\s+)?['"]?([\w$]+)['"]?\??\s*:`)

// topLevel reads the field names of the object literal type that body
// starts with, skipping comments and nested types.
func topLevel(body string) []string {
	var names []string
	depth := 0
	line := strings.Builder{}

	for i := 0; i < len(body); i++ {
		switch {
		case strings.HasPrefix(body[i:], "/*"):
			end := strings.Index(body[i:], "*/")
			if end < 0 {
				return names
			}
			i += end + 1
			continue
		case strings.HasPrefix(body[i:], "//"):
			end := strings.Index(body[i:], "\n")
			if end < 0 {
				return names
			}
			i += end - 1
			continue
		}

		c := body[i]
		switch c {
		case '{', '(', '<', '[':
			if depth == 1 && c == '{' {
				// A nested object type's opening brace still ends a field's name.
				line.WriteByte(c)
			}
			depth++
			continue
		case '}', ')', '>', ']':
			depth--
			if depth == 0 {
				// The last field may end at the brace, with no ; of its own.
				if match := fieldName.FindStringSubmatch(line.String()); match != nil {
					names = append(names, match[1])
				}
				return names
			}
			continue
		}

		if depth != 1 {
			continue
		}
		if c == ';' || c == '\n' {
			if match := fieldName.FindStringSubmatch(line.String()); match != nil {
				names = append(names, match[1])
			}
			line.Reset()
			continue
		}
		line.WriteByte(c)
	}

	return names
}
