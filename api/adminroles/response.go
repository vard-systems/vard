package adminroles

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
)

// roleResponse is one admin role as the panel sees it.
type roleResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	// Permissions are the catalog names the role grants. super_admin lists
	// every one, which is what it means.
	Permissions []string `json:"permissions"`
	// IsBuiltin marks super_admin, which cannot be changed or removed.
	IsBuiltin bool `json:"is_builtin"`
	// AdminCount is how many administrators hold the role.
	AdminCount int64     `json:"admin_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func newRoleResponse(role model.AdminRole, admins int64) roleResponse {
	permissions := []string{}
	for _, name := range model.AdminPermissionNames() {
		if role.Grants(name) {
			permissions = append(permissions, name)
		}
	}

	return roleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
		Permissions: permissions,
		IsBuiltin:   role.IsSuperAdmin(),
		AdminCount:  admins,
		CreatedAt:   role.CreatedAt,
		UpdatedAt:   role.UpdatedAt,
	}
}

// listResponse is every admin role matching a search.
type listResponse struct {
	Roles []roleResponse `json:"roles"`
	Total int            `json:"total"`
}

func newListResponse(roles []model.AdminRole, counts map[uuid.UUID]int64) listResponse {
	out := make([]roleResponse, 0, len(roles))
	for _, role := range roles {
		out = append(out, newRoleResponse(role, counts[role.ID]))
	}

	return listResponse{Roles: out, Total: len(out)}
}
