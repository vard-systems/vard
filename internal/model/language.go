package model

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Language struct {
	Base

	Code string `gorm:"size:16;not null;uniqueIndex" json:"code"`

	Name       string `gorm:"size:64;not null" json:"name"`
	NativeName string `gorm:"size:64;not null" json:"native_name"`

	IsEnabled bool `gorm:"not null" json:"is_enabled"`

	IsDefault bool `gorm:"not null" json:"is_default"`

	Position int `gorm:"not null" json:"position"`

	Translations []Translation `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

func (Language) TableName() string {
	return "languages"
}

const BaseLanguage = "en"

func DefaultLanguage() Language {
	return Language{
		Code: BaseLanguage, Name: "English", NativeName: "English",
		IsEnabled: true, IsDefault: true, Position: 1,
	}
}

// languageCodePattern is a narrow IETF tag (language, optional script or
// region), since codes go into URLs, cookies and file names.
var languageCodePattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

func (l Language) Validate() error {
	switch {
	case strings.TrimSpace(l.Code) == "":
		return fmt.Errorf("code is required")
	case len(l.Code) > 16:
		return fmt.Errorf("code must be at most 16 characters")
	case !languageCodePattern.MatchString(l.Code):
		return fmt.Errorf("code must be a language tag, such as en, ky or pt-BR")
	case strings.TrimSpace(l.Name) == "":
		return fmt.Errorf("name is required")
	case utf8.RuneCountInString(l.Name) > 64:
		return fmt.Errorf("name must be at most 64 characters")
	case strings.TrimSpace(l.NativeName) == "":
		return fmt.Errorf("native name is required")
	case utf8.RuneCountInString(l.NativeName) > 64:
		return fmt.Errorf("native name must be at most 64 characters")
	case l.Position < 0:
		return fmt.Errorf("position cannot be negative")
	}

	if l.IsDefault && !l.IsEnabled {
		return fmt.Errorf("the default language cannot be turned off")
	}

	return nil
}

type Translation struct {
	Base

	LanguageID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_translations_language_app,priority:1" json:"language_id"`
	App        string    `gorm:"size:16;not null;uniqueIndex:idx_translations_language_app,priority:2" json:"app"`

	Messages map[string]string `gorm:"type:jsonb;serializer:json;not null" json:"messages"`
}

func (Translation) TableName() string {
	return "translations"
}

const MaxMessageLength = 2000

func TooLongMessage(messages map[string]string) string {
	for key, value := range messages {
		if utf8.RuneCountInString(value) > MaxMessageLength {
			return key
		}
	}

	return ""
}
