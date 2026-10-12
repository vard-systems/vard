package model

import (
	"regexp"
	"slices"
	"sort"

	"github.com/google/uuid"
)

// UserRole is a role users hold: global (no application) or belonging to one
// application, where its name means nothing elsewhere. Names are unique within
// their scope. A role carries no permissions; applications decide what it
// allows.
//
// A role may inherit others. A global role may include global and any
// application's roles; an application role may include global roles and its own
// application's. A default role is given to every new user.
type UserRole struct {
	Base

	// ApplicationID is the application the role belongs to, or nil for a
	// global role.
	ApplicationID *uuid.UUID   `gorm:"type:uuid;uniqueIndex:idx_user_roles_application_name,priority:1" json:"application_id"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE" json:"-"`

	Name        string `gorm:"size:64;not null;uniqueIndex:idx_user_roles_application_name,priority:2" json:"name"`
	Description string `gorm:"size:255" json:"description"`
	IsDefault   bool   `gorm:"not null;default:false;index" json:"is_default"`

	Inherits []UserRole `gorm:"many2many:user_role_inherits;joinForeignKey:RoleID;joinReferences:InheritedRoleID;constraint:OnDelete:CASCADE" json:"inherits,omitempty"`

	// APIScopes are the API scopes holding this role grants. A token for an
	// API carries one only when the application is allowed it too.
	APIScopes []APIScope `gorm:"many2many:user_role_api_scopes;constraint:OnDelete:CASCADE" json:"api_scopes,omitempty"`
}

func (r UserRole) Global() bool {
	return r.ApplicationID == nil
}

func (r UserRole) In(application *uuid.UUID) bool {
	if r.ApplicationID == nil || application == nil {
		return r.ApplicationID == nil && application == nil
	}

	return *r.ApplicationID == *application
}

func (r UserRole) MayInherit(other UserRole) bool {
	return r.Global() || other.Global() || other.In(r.ApplicationID)
}

func (UserRole) TableName() string {
	return "user_roles"
}

var RoleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

type RoleGraph map[uuid.UUID]UserRole

func (g RoleGraph) Reaches(from, to uuid.UUID) bool {
	for _, role := range g.walk([]uuid.UUID{from}) {
		if role.ID == to {
			return true
		}
	}

	return false
}

func (g RoleGraph) WouldCycle(role uuid.UUID, inherits []uuid.UUID) bool {
	for _, id := range inherits {
		if id == role || g.Reaches(id, role) {
			return true
		}
	}

	return false
}

func (g RoleGraph) Effective(held []uuid.UUID) []UserRole {
	roles := g.walk(held)
	sort.Slice(roles, func(i, j int) bool { return roles[i].Name < roles[j].Name })

	return roles
}

// walk returns every role reached from these ids, each once, in no particular
// order. A cycle stored by some other route does not make it loop.
func (g RoleGraph) walk(from []uuid.UUID) []UserRole {
	seen := map[uuid.UUID]bool{}
	stack := append([]uuid.UUID(nil), from...)

	var roles []UserRole
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		role, ok := g[id]
		if !ok || seen[id] {
			continue
		}
		seen[id] = true

		roles = append(roles, role)
		for _, inherited := range role.Inherits {
			stack = append(stack, inherited.ID)
		}
	}

	return roles
}

func (g RoleGraph) Via(held []uuid.UUID) map[uuid.UUID][]uuid.UUID {
	via := map[uuid.UUID][]uuid.UUID{}

	for _, start := range held {
		for _, reached := range g.walk([]uuid.UUID{start}) {
			if reached.ID == start {
				continue
			}
			if !slices.Contains(via[reached.ID], start) {
				via[reached.ID] = append(via[reached.ID], start)
			}
		}
	}

	return via
}

func Names(roles []UserRole) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Name)
	}

	return names
}
