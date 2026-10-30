package roles

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/infrastructure/store"
)

// roleRef names another role, and its scope: enough to show it and to send
// it back.
type roleRef struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	ApplicationID *uuid.UUID `json:"application_id"`
}

func newRoleRef(role model.UserRole) roleRef {
	return roleRef{ID: role.ID, Name: role.Name, ApplicationID: role.ApplicationID}
}

// scopeRef names an API scope, and the API it belongs to.
type scopeRef struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	APIID uuid.UUID `json:"api_id"`
}

// roleResponse is one role as the panel sees it.
type roleResponse struct {
	ID uuid.UUID `json:"id"`
	// ApplicationID is the application the role belongs to, or null for a
	// global role.
	ApplicationID *uuid.UUID `json:"application_id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	IsDefault     bool       `json:"is_default"`

	// Inherits are the roles this one includes directly.
	Inherits []roleRef `json:"inherits"`

	// InheritedRoles is every role this one includes, transitively, sorted by
	// name; roles of invisible applications are left out.
	InheritedRoles []roleRef `json:"inherited_roles"`

	// APIScopes are the API scopes the role grants directly.
	APIScopes []scopeRef `json:"api_scopes"`

	// UserCount is how many users hold the role directly.
	UserCount int64 `json:"user_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// roleDetails is what a response needs beyond the role itself.
type roleDetails struct {
	members map[uuid.UUID]int64
	graph   model.RoleGraph
	// visible says whether the administrator can see a role.
	visible func(model.UserRole) bool
}

func newRoleResponse(role model.UserRole, details roleDetails) roleResponse {
	inherits := make([]roleRef, 0, len(role.Inherits))
	for _, inherited := range role.Inherits {
		if details.visible(inherited) {
			inherits = append(inherits, newRoleRef(inherited))
		}
	}

	included := []roleRef{}
	for _, reached := range details.graph.Effective([]uuid.UUID{role.ID}) {
		if reached.ID != role.ID && details.visible(reached) {
			included = append(included, newRoleRef(reached))
		}
	}

	scopes := make([]scopeRef, 0, len(role.APIScopes))
	for _, scope := range role.APIScopes {
		scopes = append(scopes, scopeRef{ID: scope.ID, Name: scope.Name, APIID: scope.APIID})
	}

	return roleResponse{
		APIScopes:      scopes,
		ID:             role.ID,
		ApplicationID:  role.ApplicationID,
		Name:           role.Name,
		Description:    role.Description,
		IsDefault:      role.IsDefault,
		Inherits:       inherits,
		InheritedRoles: included,
		UserCount:      details.members[role.ID],
		CreatedAt:      role.CreatedAt,
		UpdatedAt:      role.UpdatedAt,
	}
}

// pageResponse is a page of roles, with enough about the page for the panel
// to say how many there are without asking again.
type pageResponse struct {
	Roles  []roleResponse `json:"roles"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func newPageResponse(roles []model.UserRole, details roleDetails, total int64, query store.RoleQuery) pageResponse {
	out := make([]roleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, newRoleResponse(role, details))
	}

	return pageResponse{
		Roles:  out,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	}
}
