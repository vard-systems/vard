package model

import (
	"slices"

	"github.com/google/uuid"
)

// Consent is what a user has allowed an application on the consent page. A
// request for nothing more goes through without asking again; one for more
// asks for all of it.
type Consent struct {
	Base

	UserID        uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_consents_pair"`
	User          *User        `gorm:"constraint:OnDelete:CASCADE"`
	ApplicationID uuid.UUID    `gorm:"type:uuid;not null;uniqueIndex:idx_consents_pair;index"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE"`

	Scopes []string `gorm:"type:jsonb;serializer:json;not null"`
}

func (Consent) TableName() string {
	return "consents"
}

// Covers says every scope asked for has been allowed before.
func (c Consent) Covers(scopes []string) bool {
	for _, scope := range scopes {
		if !slices.Contains(c.Scopes, scope) {
			return false
		}
	}

	return true
}

// Allow adds scopes to what has been allowed, keeping what was.
func (c *Consent) Allow(scopes []string) {
	for _, scope := range scopes {
		if !slices.Contains(c.Scopes, scope) {
			c.Scopes = append(c.Scopes, scope)
		}
	}
}
