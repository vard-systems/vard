package roles

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/query"
	"loginer/internal/store"
)

// targetType is what these records are called in the activity log.
const targetType = "user_role"

// maxPageSize is high because the panel loads every visible role at once as
// choices.
const maxPageSize = 500

// roleRequest is the create and update body. Inherits holds ids and replaces
// the role's inheritance.
type roleRequest struct {
	// ApplicationID is the application the role belongs to, or null for a
	// global role. It is read on create only: a role keeps its scope.
	ApplicationID *uuid.UUID `json:"application_id"`

	Name        string `json:"name" validate:"required,rolename"`
	Description string `json:"description" validate:"max=255"`
	// IsDefault gives the role to every user created without roles of their
	// own.
	IsDefault bool `json:"is_default"`

	Inherits []uuid.UUID `json:"inherits"`

	// APIScopes replace the role's scope grants; left out, they are kept, so
	// administrators who cannot see APIs can still edit roles.
	APIScopes *[]uuid.UUID `json:"api_scopes"`
}

func (r *roleRequest) clean() {
	r.Name = strings.ToLower(strings.TrimSpace(r.Name))
	r.Description = strings.TrimSpace(r.Description)
}

func listQuery(c *gin.Context) store.RoleQuery {
	return store.RoleQuery{
		Search:  c.Query("search"),
		Limit:   query.Int(c, "limit", 50, maxPageSize),
		Offset:  query.Int(c, "offset", 0, query.MaxOffset),
		Default: query.Bool(c, "default"),
	}
}
