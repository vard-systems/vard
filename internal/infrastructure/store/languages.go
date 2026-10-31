package store

import (
	"context"
	"errors"
	"maps"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"vard/i18n"
	"vard/internal/domain/model"
	"vard/internal/infrastructure/cache"
)

var ErrProtectedLanguage = errors.New("this language cannot be removed")

func (s *Store) Languages(ctx context.Context) ([]model.Language, error) {
	return cached(ctx, s, cache.Languages, "all", func() ([]model.Language, error) {
		var languages []model.Language
		err := s.db.WithContext(ctx).Order("position, code").Find(&languages).Error

		return languages, err
	})
}

func (s *Store) Language(ctx context.Context, code string) (*model.Language, error) {
	return cached(ctx, s, cache.Languages, "code:"+code, func() (*model.Language, error) {
		var language model.Language
		if err := s.db.WithContext(ctx).First(&language, "code = ?", code).Error; err != nil {
			return nil, translate(err)
		}

		return &language, nil
	})
}

func (s *Store) DefaultLanguage(ctx context.Context) (*model.Language, error) {
	var found model.Language
	if s.cache.Get(ctx, cache.Languages, "default", &found) {
		return &found, nil
	}

	var language model.Language

	err := translate(s.db.WithContext(ctx).Order("position, code").First(&language, "is_default = ?", true).Error)
	switch {
	case err == nil:
		s.cache.Set(ctx, cache.Languages, "default", language)
		return &language, nil
	case !errors.Is(err, ErrNotFound):
		return nil, err
	}

	language = model.DefaultLanguage()
	if err := s.db.WithContext(ctx).Create(&language).Error; err != nil {
		return nil, translate(err)
	}

	s.forget(ctx, cache.Languages)

	return &language, nil
}

func (s *Store) OfferedLanguages(ctx context.Context) ([]model.Language, error) {
	if _, err := s.DefaultLanguage(ctx); err != nil {
		return nil, err
	}

	return cached(ctx, s, cache.Languages, "offered", func() ([]model.Language, error) {
		var languages []model.Language
		err := s.db.WithContext(ctx).Where("is_enabled = ?", true).Order("position, code").Find(&languages).Error

		return languages, err
	})
}

func (s *Store) CreateLanguage(ctx context.Context, language *model.Language, messages map[string]map[string]string) error {
	language.Code = strings.TrimSpace(language.Code)

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(language).Error; err != nil {
			return translate(err)
		}

		for app, text := range messages {
			if err := saveTranslation(tx, language.ID, app, text); err != nil {
				return err
			}
		}

		return demoteOtherLanguages(tx, language)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.Languages)

	return nil
}

func (s *Store) SaveLanguage(ctx context.Context, language *model.Language) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(language).Error; err != nil {
			return translate(err)
		}

		return demoteOtherLanguages(tx, language)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.Languages)

	return nil
}

func (s *Store) DeleteLanguage(ctx context.Context, language *model.Language) error {
	if language.Code == model.BaseLanguage || language.IsDefault {
		return ErrProtectedLanguage
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Hard delete: a soft-deleted row would keep the unique code taken.
		if err := tx.Where("language_id = ?", language.ID).Delete(&model.Translation{}).Error; err != nil {
			return translate(err)
		}

		return translate(tx.Delete(language).Error)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.Languages)

	return nil
}

func demoteOtherLanguages(tx *gorm.DB, language *model.Language) error {
	if !language.IsDefault {
		return nil
	}

	err := tx.Model(&model.Language{}).
		Where("id <> ? AND is_default = ?", language.ID, true).
		Update("is_default", false).Error

	return translate(err)
}

func (s *Store) NextLanguagePosition(ctx context.Context) int {
	return nextPosition(s.db.WithContext(ctx))
}

func (s *Store) Translations(ctx context.Context, languages ...uuid.UUID) (map[uuid.UUID]map[string]map[string]string, error) {
	out := map[uuid.UUID]map[string]map[string]string{}
	if len(languages) == 0 {
		return out, nil
	}

	var rows []model.Translation
	if err := s.db.WithContext(ctx).Where("language_id IN ?", languages).Find(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		if out[row.LanguageID] == nil {
			out[row.LanguageID] = map[string]map[string]string{}
		}
		out[row.LanguageID][row.App] = row.Messages
	}

	return out, nil
}

func (s *Store) Translation(ctx context.Context, language uuid.UUID, app string) (map[string]string, error) {
	var row model.Translation

	err := translate(s.db.WithContext(ctx).First(&row, "language_id = ? AND app = ?", language, app).Error)
	switch {
	case errors.Is(err, ErrNotFound):
		return map[string]string{}, nil
	case err != nil:
		return nil, err
	}

	return row.Messages, nil
}

func (s *Store) ResolvedTranslation(ctx context.Context, language *model.Language, app i18n.App) (map[string]string, error) {
	return cached(ctx, s, cache.Languages, "text:"+language.Code+":"+string(app), func() (map[string]string, error) {
		return s.resolveTranslation(ctx, language, app)
	})
}

func (s *Store) resolveTranslation(ctx context.Context, language *model.Language, app i18n.App) (map[string]string, error) {
	own, err := s.Translation(ctx, language.ID, string(app))
	if err != nil {
		return nil, err
	}

	if language.Code == model.BaseLanguage {
		return i18n.Resolve(app, own), nil
	}

	base, err := s.Language(ctx, model.BaseLanguage)
	switch {
	case errors.Is(err, ErrNotFound):
		return i18n.Resolve(app, own), nil
	case err != nil:
		return nil, err
	}

	english, err := s.Translation(ctx, base.ID, string(app))
	if err != nil {
		return nil, err
	}

	return i18n.Resolve(app, own, english), nil
}

func (s *Store) SaveTranslation(ctx context.Context, language *model.Language, app string, messages map[string]string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := saveTranslation(tx, language.ID, app, messages); err != nil {
			return err
		}

		return translate(tx.Model(language).Update("updated_at", time.Now()).Error)
	})
	if err != nil {
		return err
	}

	// Every language's text, not only this one's: English is what the rest
	// fall back to, so its keys are inside every other cached text.
	s.forget(ctx, cache.Languages)

	return nil
}

func (s *Store) SaveTranslationKeys(ctx context.Context, language *model.Language, app string, messages map[string]string) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.Translation

		err := translate(tx.First(&row, "language_id = ? AND app = ?", language.ID, app).Error)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}

		merged := map[string]string{}
		maps.Copy(merged, row.Messages)
		for key, text := range messages {
			if text == "" {
				delete(merged, key)
				continue
			}
			merged[key] = text
		}

		if err := saveTranslation(tx, language.ID, app, merged); err != nil {
			return err
		}

		return translate(tx.Model(language).Update("updated_at", time.Now()).Error)
	})
	if err != nil {
		return err
	}

	s.forget(ctx, cache.Languages)

	return nil
}

func saveTranslation(tx *gorm.DB, language uuid.UUID, app string, messages map[string]string) error {
	if messages == nil {
		messages = map[string]string{}
	}

	row := model.Translation{LanguageID: language, App: app, Messages: messages}

	err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "language_id"}, {Name: "app"}},
		DoUpdates: clause.AssignmentColumns([]string{"messages", "updated_at"}),
	}).Create(&row).Error

	return translate(err)
}

func nextPosition(tx *gorm.DB) int {
	var last model.Language
	if err := tx.Order("position DESC").First(&last).Error; err != nil {
		return 1
	}

	return last.Position + 1
}
