package adminroles

import "strings"

const targetType = "admin_role"

// roleRequest is the body of the create and update endpoints. Permissions are
// names from the catalog, and replace what the role granted.
type roleRequest struct {
	Name        string   `json:"name" validate:"required,adminrole"`
	Description string   `json:"description" validate:"max=255"`
	Permissions []string `json:"permissions"`
}

func (r *roleRequest) clean() {
	r.Name = strings.ToLower(strings.TrimSpace(r.Name))
	r.Description = strings.TrimSpace(r.Description)

	for i, permission := range r.Permissions {
		r.Permissions[i] = strings.TrimSpace(permission)
	}
}
