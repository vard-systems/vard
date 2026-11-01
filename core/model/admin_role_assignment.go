package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AdminRoleAssignment struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`

	AdminID uuid.UUID `gorm:"type:uuid;not null;index" json:"admin_id"`
	Admin   *Admin    `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	RoleID uuid.UUID `gorm:"type:uuid;not null;index" json:"role_id"`
	Role   AdminRole `gorm:"constraint:OnDelete:CASCADE" json:"role"`

	ApplicationID *uuid.UUID   `gorm:"type:uuid;index" json:"application_id"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE" json:"application,omitempty"`
}

func (AdminRoleAssignment) TableName() string {
	return "admin_role_assignments"
}

func (a *AdminRoleAssignment) BeforeCreate(*gorm.DB) error {
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

func (a AdminRoleAssignment) Global() bool {
	return a.ApplicationID == nil
}

func (a AdminRoleAssignment) Grants(permission string, application *uuid.UUID) bool {
	if a.Global() {
		return a.Role.Grants(permission)
	}

	return application != nil &&
		*a.ApplicationID == *application &&
		IsScopablePermission(permission) &&
		a.Role.Grants(permission)
}
