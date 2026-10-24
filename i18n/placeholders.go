package i18n

import (
	"fmt"
	"regexp"

	"github.com/nicksnyder/go-i18n/v2/i18n/template"
)

// placeholders is go-i18n's template parser for ICU's `{name}`, so the text
// svelte-i18n formats in the browser is the text formatted here. A name the
// data lacks is left as written.
var placeholders template.Parser = placeholderParser{}

var placeholder = regexp.MustCompile(`\{(\w+)\}`)

type placeholderParser struct{}

func (placeholderParser) Cacheable() bool { return true }

func (placeholderParser) Parse(src, _, _ string) (template.ParsedTemplate, error) {
	return placeholderTemplate(src), nil
}

type placeholderTemplate string

func (t placeholderTemplate) Execute(data any) (string, error) {
	params, _ := data.(map[string]any)
	if len(params) == 0 {
		return string(t), nil
	}

	return placeholder.ReplaceAllStringFunc(string(t), func(whole string) string {
		value, ok := params[whole[1:len(whole)-1]]
		if !ok {
			return whole
		}

		return fmt.Sprint(value)
	}), nil
}
