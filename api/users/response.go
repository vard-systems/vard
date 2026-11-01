package users

import (
	"slices"
	"strings"

	"github.com/google/uuid"

	"vard/core/model"
	"vard/core/store"
)

// pageResponse is a page of users, with enough about the page for the panel
// to say "1 of 4" without asking again.
type pageResponse struct {
	Users  []model.User `json:"users"`
	Total  int64        `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

func newPageResponse(users []model.User, total int64, query store.UserQuery) pageResponse {
	return pageResponse{
		Users:  users,
		Total:  total,
		Limit:  query.Limit,
		Offset: query.Offset,
	}
}

// heldRoles is what a user holds in one scope.
type heldRoles struct {
	// Roles are the roles given to the user directly.
	Roles []string `json:"roles"`
	// EffectiveRoles are those plus every role of the scope they include,
	// sorted by name. This is the list a token carries.
	EffectiveRoles []string `json:"effective_roles"`
}

// applicationRoles is what a user holds in one application.
type applicationRoles struct {
	ID       uuid.UUID `json:"id"`
	ClientID string    `json:"client_id"`
	Name     string    `json:"name"`
	heldRoles
}

// rolesResponse is the roles a user holds as tokens would carry them, listing
// every application even when empty.
type rolesResponse struct {
	Global       heldRoles          `json:"global"`
	Applications []applicationRoles `json:"applications"`
}

func newRolesResponse(user *model.User, apps []model.Application, graph model.RoleGraph) rolesResponse {
	held := make([]uuid.UUID, 0, len(user.Roles))
	for _, role := range user.Roles {
		held = append(held, role.ID)
	}
	effective := graph.Effective(held)

	scope := func(app *uuid.UUID) heldRoles {
		direct := slices.DeleteFunc(slices.Clone(user.Roles), func(role model.UserRole) bool { return !role.In(app) })
		reached := slices.DeleteFunc(slices.Clone(effective), func(role model.UserRole) bool { return !role.In(app) })

		return heldRoles{Roles: model.Names(direct), EffectiveRoles: model.Names(reached)}
	}

	out := rolesResponse{Global: scope(nil), Applications: make([]applicationRoles, 0, len(apps))}
	for _, app := range apps {
		out.Applications = append(out.Applications, applicationRoles{
			ID:        app.ID,
			ClientID:  app.ClientID,
			Name:      app.Name,
			heldRoles: scope(&app.ID),
		})
	}

	return out
}

// appRef names an application a role belongs to.
type appRef struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	ClientID string    `json:"client_id"`
}

type roleRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// mappedRole is one role a user holds, as the role mapping tab lists it.
type mappedRole struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	// Application is the application the role belongs to, or null for a
	// global role.
	Application *appRef `json:"application"`
	// Composite says the role includes other roles.
	Composite bool `json:"composite"`

	// Assigned means given directly; Via lists the direct roles it also comes
	// through. With both, removing it leaves the user holding it.
	Assigned bool      `json:"assigned"`
	Via      []roleRef `json:"via"`
}

// mappingsResponse is every role a user holds: global roles first, then each
// application's, each sorted by name.
type mappingsResponse struct {
	Roles []mappedRole `json:"roles"`
}

func newMappingsResponse(
	user *model.User,
	apps []model.Application,
	graph model.RoleGraph,
	visible func(model.UserRole) bool,
) mappingsResponse {
	held := make([]uuid.UUID, 0, len(user.Roles))
	for _, role := range user.Roles {
		held = append(held, role.ID)
	}

	names := make(map[uuid.UUID]model.Application, len(apps))
	for _, app := range apps {
		names[app.ID] = app
	}

	via := graph.Via(held)
	out := mappingsResponse{Roles: []mappedRole{}}

	for _, role := range graph.Effective(held) {
		if !visible(role) {
			continue
		}

		mapped := mappedRole{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			Composite:   len(role.Inherits) > 0,
			Assigned:    slices.Contains(held, role.ID),
			Via:         []roleRef{},
		}

		if role.ApplicationID != nil {
			app := names[*role.ApplicationID]
			mapped.Application = &appRef{ID: *role.ApplicationID, Name: app.Name, ClientID: app.ClientID}
		}

		for _, through := range via[role.ID] {
			if parent, ok := graph[through]; ok && visible(parent) {
				mapped.Via = append(mapped.Via, roleRef{ID: parent.ID, Name: parent.Name})
			}
		}

		out.Roles = append(out.Roles, mapped)
	}

	slices.SortStableFunc(out.Roles, func(a, b mappedRole) int {
		return strings.Compare(sortKey(a), sortKey(b))
	})

	return out
}

func sortKey(role mappedRole) string {
	if role.Application == nil {
		return "0\x00" + role.Name
	}

	return "1\x00" + strings.ToLower(role.Application.Name) + "\x00" + role.Name
}
