package model

import "time"

// SetupTokenLifetime is how long the token a server logs at startup can set
// the panel up; a server started again while nobody has logs another.
const SetupTokenLifetime = 24 * time.Hour

// SetupToken is what creating the first administrator takes: whoever can read
// the server's log, not whoever reaches the panel first. Only its hash is
// kept, and it is taken once.
type SetupToken struct {
	Base

	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null;index"`
	UsedAt    *time.Time
}

func (SetupToken) TableName() string {
	return "setup_tokens"
}
