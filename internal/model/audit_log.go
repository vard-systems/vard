package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog is one recorded action. The table is append-only: rows are never
// updated or deleted, so it does not embed Base.
type AuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`

	// AdminID is the admin who acted, and is null for actions the system
	// took on its own. It is not a foreign key: the log outlives the account.
	AdminID    *uuid.UUID `gorm:"type:uuid;index" json:"admin_id,omitempty"`
	ActorEmail string     `gorm:"size:255" json:"actor_email,omitempty"`

	Action string `gorm:"size:128;not null" json:"action"`

	TargetType string `gorm:"index;size:64" json:"target_type,omitempty"`
	TargetID   string `gorm:"index;size:64" json:"target_id,omitempty"`

	// Metadata carries whatever else is worth keeping, such as the fields that
	// changed. Never put secrets or passwords in it.
	Metadata map[string]any `gorm:"serializer:json" json:"metadata,omitempty"`

	IP        string `gorm:"size:45" json:"ip,omitempty"`
	UserAgent string `gorm:"size:255" json:"user_agent,omitempty"`
	RequestID string `gorm:"index;size:64" json:"request_id,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

func (a *AuditLog) BeforeCreate(*gorm.DB) error {
	if a.ID != uuid.Nil {
		return nil
	}

	id, err := newID()
	if err != nil {
		return err
	}
	a.ID = id

	return nil
}
