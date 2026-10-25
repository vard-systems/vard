package oidc

import (
	"context"

	"vard/i18n"
	"vard/internal/model"
	"vard/internal/store"
)

// PublicLanguage is one language in the picker: its code and its English and
// native names.
type PublicLanguage struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	NativeName string `json:"native_name"`
}

func publicLanguage(language model.Language) PublicLanguage {
	return PublicLanguage{Code: language.Code, Name: language.Name, NativeName: language.NativeName}
}

func (s *Service) Languages(ctx context.Context) ([]PublicLanguage, string, error) {
	offered, err := s.store.OfferedLanguages(ctx)
	if err != nil {
		return nil, "", err
	}

	fallback, err := s.store.DefaultLanguage(ctx)
	if err != nil {
		return nil, "", err
	}

	languages := make([]PublicLanguage, 0, len(offered))
	for _, language := range offered {
		languages = append(languages, publicLanguage(language))
	}

	return languages, fallback.Code, nil
}

// textIn is the sign-in pages' text in the language code names, or the
// default when it is not offered, or the shipped English when the database
// cannot say.
func (s *Service) textIn(ctx context.Context, code string) *i18n.Catalog {
	language, err := s.store.Language(ctx, code)
	if err != nil || !language.IsEnabled {
		language, err = s.store.DefaultLanguage(ctx)
	}

	if err == nil {
		if text, err := s.store.ResolvedTranslation(ctx, language, i18n.ID); err == nil {
			return i18n.NewCatalog(language.Code, text)
		}
	}

	return i18n.English(i18n.ID)
}

func (s *Service) LanguageText(ctx context.Context, code string) (PublicLanguage, map[string]string, error) {
	language, err := s.store.Language(ctx, code)
	if err != nil {
		return PublicLanguage{}, nil, err
	}

	if !language.IsEnabled {
		return PublicLanguage{}, nil, store.ErrNotFound
	}

	messages, err := s.store.ResolvedTranslation(ctx, language, i18n.ID)
	if err != nil {
		return PublicLanguage{}, nil, err
	}

	return publicLanguage(*language), messages, nil
}
