package model

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Admin struct {
	Base

	Username     string `gorm:"uniqueIndex;size:100;not null" json:"username"`
	Email        string `gorm:"uniqueIndex;size:255;not null" json:"email"`
	FirstName    string `gorm:"size:100;not null" json:"first_name"`
	LastName     string `gorm:"size:100;not null" json:"last_name"`
	AvatarURL    string `gorm:"size:512;not null;default:''" json:"avatar_url"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	IsActive     bool   `gorm:"-" json:"is_active"`
	Status       Status `gorm:"type:varchar(32);index;not null;default:invited" json:"status"`

	EmailVerifiedAt *time.Time `json:"email_verified_at,omitempty"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP     string     `gorm:"size:45;not null;default:''" json:"last_login_ip,omitempty"`

	Service bool `gorm:"-" json:"-"`

	Assignments []AdminRoleAssignment `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	Sessions    []AdminSession        `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	MFA         []MFA                 `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

// TableName pins the table name so renaming the struct cannot silently rename
// the table.
func (Admin) TableName() string {
	return "admins"
}

func (a *Admin) SetPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	a.PasswordHash = string(hash)

	return nil
}

func (a Admin) FullName() string {
	return strings.TrimSpace(a.FirstName + " " + a.LastName)
}

func (a Admin) CanSignIn() bool {
	return a.Status == StatusActive
}

func (a Admin) HasRole(name string) bool {
	for _, assignment := range a.Assignments {
		if assignment.Global() && assignment.Role.Name == name {
			return true
		}
	}
	return false
}

func (a Admin) IsSuperAdmin() bool {
	return a.HasRole(RoleSuperAdmin)
}

func (a Admin) HasPermission(name string) bool {
	return a.HasPermissionFor(name, nil)
}

func (a Admin) HasPermissionFor(name string, application *uuid.UUID) bool {
	for _, assignment := range a.Assignments {
		if assignment.Grants(name, application) {
			return true
		}
	}
	return false
}

func (a Admin) HasPermissionAnywhere(name string) bool {
	if a.HasPermission(name) {
		return true
	}

	for _, assignment := range a.Assignments {
		if !assignment.Global() && assignment.Grants(name, assignment.ApplicationID) {
			return true
		}
	}
	return false
}

func (a Admin) ApplicationsWith(name string) (all bool, ids []uuid.UUID) {
	if a.HasPermission(name) {
		return true, nil
	}

	ids = []uuid.UUID{}
	for _, assignment := range a.Assignments {
		if !assignment.Global() && assignment.Grants(name, assignment.ApplicationID) {
			ids = append(ids, *assignment.ApplicationID)
		}
	}
	return false, ids
}

func (a Admin) Permissions() []string {
	granted := []string{}
	for _, name := range AdminPermissionNames() {
		if a.HasPermission(name) {
			granted = append(granted, name)
		}
	}
	return granted
}

func (a Admin) ScopedPermissions() map[uuid.UUID][]string {
	scoped := map[uuid.UUID][]string{}

	for _, assignment := range a.Assignments {
		if assignment.Global() {
			continue
		}

		app := *assignment.ApplicationID
		for _, name := range AdminPermissionNames() {
			if a.HasPermission(name) || !assignment.Grants(name, &app) {
				continue
			}
			if !slices.Contains(scoped[app], name) {
				scoped[app] = append(scoped[app], name)
			}
		}
	}

	return scoped
}

// ServiceAdmin is an application calling the admin API with a token, presented
// as an administrator holding exactly its token's permissions and never a super
// admin. Username is its client ID, which the activity log shows.
func ServiceAdmin(app Application, permissions []string) *Admin {
	return &Admin{
		Username:  app.ClientID,
		FirstName: app.Name,
		IsActive:  true,
		Service:   true,
		Status:    StatusActive,
		Assignments: []AdminRoleAssignment{{
			Role: AdminRole{Name: "application:" + app.ClientID, Permissions: permissions},
		}},
	}
}
