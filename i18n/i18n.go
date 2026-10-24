// Package i18n holds the English the server ships and formats text in any
// language with go-i18n.
//
// English is the only language that ships: `i18n/id/*.en.json` are the
// sign-in pages, `i18n/console/*.en.json` the admin API's problems. Every
// other language is added, translated and offered from the panel and lives in
// the database; its text is resolved against these files (Resolve) and
// formatted through a Catalog.
//
// Placeholders are ICU's `{name}`, the same text svelte-i18n formats on the
// sign-in pages, so one translation serves both.
package i18n

import (
	"embed"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"
	"sync"

	goi18n "github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed id/*.en.json console/*.en.json
var files embed.FS

type App string

const (
	// ID is the sign-in pages: the one app a language is translated for.
	ID App = "id"
	// Console is the admin API's English; the panel itself is not translated.
	Console App = "console"
)

// Apps are the apps a language added in the panel is translated for.
var Apps = []App{ID}

var catalogued = []App{ID, Console}

func ParseApp(name string) (App, bool) {
	for _, app := range Apps {
		if string(app) == name {
			return app, true
		}
	}

	return "", false
}

// Base is the language every other falls back to.
const Base = "en"

type Messages = map[string]string

// Keys are the keys an app has, sorted.
func Keys(app App) []string {
	return load().keys[app]
}

// Text is the shipped English of one key, unformatted.
func Text(app App, key string) (string, bool) {
	text, ok := load().english[app][key]

	return text, ok
}

// English is the shipped English of an app as a Catalog.
func English(app App) *Catalog {
	return load().catalogs[app]
}

// Coverage is how much of an app's keys messages translates.
func Coverage(app App, messages Messages) (percent, missing int) {
	keys := Keys(app)
	if len(keys) == 0 {
		return 100, 0
	}

	translated := 0
	for _, key := range keys {
		if strings.TrimSpace(messages[key]) != "" {
			translated++
		}
	}

	return translated * 100 / len(keys), len(keys) - translated
}

// Resolve is every key of an app, from the first layer that has it, else the
// shipped English.
func Resolve(app App, layers ...Messages) Messages {
	english := load().english[app]
	out := make(Messages, len(english))

	for key, text := range english {
		out[key] = text

		for _, layer := range layers {
			if value := layer[key]; strings.TrimSpace(value) != "" {
				out[key] = value
				break
			}
		}
	}

	return out
}

// Known keeps the keys an app has and are filled in, and counts the ones it
// does not have. A file the panel exports names its language in `$name` and
// `$native`, which are passed over without being counted.
func Known(app App, messages Messages) (Messages, int) {
	english := load().english[app]
	out := make(Messages, len(messages))
	dropped := 0

	for key, value := range messages {
		switch {
		case strings.HasPrefix(key, "$"):
		case english[key] == "":
			dropped++
		case strings.TrimSpace(value) != "":
			out[key] = value
		}
	}

	return out, dropped
}

// Catalog is one language's text, formatted by go-i18n.
type Catalog struct {
	localizer *goi18n.Localizer
}

// NewCatalog is a catalog of messages in the language code names. A code
// go-i18n cannot read is taken as English, which only changes plural rules.
func NewCatalog(code string, messages Messages) *Catalog {
	tag, err := language.Parse(code)
	if err != nil {
		tag = language.English
	}

	bundle := goi18n.NewBundle(tag)
	for key, text := range messages {
		bundle.MustAddMessages(tag, &goi18n.Message{ID: key, Other: text})
	}

	return &Catalog{localizer: goi18n.NewLocalizer(bundle, tag.String())}
}

// Has says whether the catalog has a key.
func (c *Catalog) Has(key string) bool {
	_, err := c.localizer.Localize(&goi18n.LocalizeConfig{MessageID: key, TemplateParser: placeholders})

	return err == nil
}

// Localize is a key's text with params filled in, or the key itself when the
// catalog lacks it.
func (c *Catalog) Localize(key string, params map[string]any) string {
	text, err := c.localizer.Localize(&goi18n.LocalizeConfig{
		MessageID:      key,
		TemplateData:   params,
		TemplateParser: placeholders,
	})
	if err != nil {
		return key
	}

	return text
}

type shipped struct {
	english  map[App]Messages
	keys     map[App][]string
	catalogs map[App]*Catalog
}

// load reads the embedded files once. They are compiled in, so one that does
// not parse is a broken build: the tests fail on it before anything ships.
var load = sync.OnceValue(func() *shipped {
	out := &shipped{english: map[App]Messages{}, keys: map[App][]string{}, catalogs: map[App]*Catalog{}}

	for _, app := range catalogued {
		english, err := readApp(app)
		if err != nil {
			panic(err)
		}

		out.english[app] = english
		out.keys[app] = slices.Sorted(maps.Keys(english))
		out.catalogs[app] = NewCatalog(Base, english)
	}

	return out
})

// readApp parses an app's files with go-i18n, which flattens nested groups to
// dotted keys. A key given in two files is refused.
func readApp(app App) (Messages, error) {
	paths, err := fs.Glob(files, string(app)+"/*."+Base+".json")
	if err != nil {
		return nil, err
	}

	bundle := goi18n.NewBundle(language.English)
	out := Messages{}

	for _, path := range paths {
		file, err := bundle.LoadMessageFileFS(files, path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}

		for _, message := range file.Messages {
			if _, twice := out[message.ID]; twice {
				return nil, fmt.Errorf("read %s: %s is given twice", path, message.ID)
			}
			out[message.ID] = message.Other
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("i18n/%s has no %s text: it is what every language falls back to", app, Base)
	}

	return out, nil
}
