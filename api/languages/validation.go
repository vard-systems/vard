package languages

import (
	"strings"

	"vard/api/kit/respond"
	"vard/api/kit/validate"
	"vard/core/model"
)

func (r *languageRequest) applyTo(language *model.Language) error {
	if err := validate.Struct(r); err != nil {
		return err
	}

	if r.Name != nil {
		language.Name = strings.TrimSpace(*r.Name)
	}
	if r.NativeName != nil {
		language.NativeName = strings.TrimSpace(*r.NativeName)
	}
	if r.Position != nil {
		language.Position = *r.Position
	}

	language.IsEnabled = validate.Flag(r.IsEnabled, language.IsEnabled)
	language.IsDefault = validate.Flag(r.IsDefault, language.IsDefault)

	return settle(language)
}

func (r *createRequest) newLanguage() (*model.Language, error) {
	r.Code = normalizeCode(r.Code)
	r.Name = strings.TrimSpace(r.Name)
	r.NativeName = strings.TrimSpace(r.NativeName)

	if err := validate.Struct(r); err != nil {
		return nil, err
	}

	language := &model.Language{
		Code:       normalizeCode(r.Code),
		Name:       strings.TrimSpace(r.Name),
		NativeName: strings.TrimSpace(r.NativeName),
		IsEnabled:  validate.Flag(r.IsEnabled, false),
		IsDefault:  validate.Flag(r.IsDefault, false),
	}

	return language, settle(language)
}

func settle(language *model.Language) error {
	if language.IsDefault {
		language.IsEnabled = true
	}

	// The base language is always offered: a page drawn in it is the last
	// thing that still works when every other language is short of a key.
	if language.Code == model.BaseLanguage {
		language.IsEnabled = true
	}

	if err := language.Validate(); err != nil {
		return respond.Invalid(err.Error())
	}

	return nil
}

func normalizeCode(code string) string {
	parts := strings.Split(strings.TrimSpace(code), "-")

	for i, part := range parts {
		switch {
		case i == 0:
			parts[i] = strings.ToLower(part)
		case len(part) == 4:
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		case len(part) == 2 || len(part) == 3 && part[0] >= '0' && part[0] <= '9':
			parts[i] = strings.ToUpper(part)
		default:
			parts[i] = strings.ToLower(part)
		}
	}

	return strings.Join(parts, "-")
}
