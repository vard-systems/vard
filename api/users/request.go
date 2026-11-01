package users

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/api/kit/query"
	"vard/core/store"
)

const targetType = "user"

const maxPageSize = 200

// maxApplications caps how many applications a user's roles are reported for.
const maxApplications = 500

// assignRequest is the body of the endpoint that gives a user roles: the ids
// of the roles to add, global and application roles alike.
type assignRequest struct {
	Roles []uuid.UUID `json:"roles"`
}

// userRequest is the body of the create and update endpoints. Named fields are
// built-in columns; Data holds additional fields, checked against user_fields
// in validation.go.
type userRequest struct {
	Email string `json:"email" validate:"required,email,max=255"`
	// IsEmailVerified and IsActive may be left out: a new user is then
	// unverified and active, and an existing one keeps what they had.
	IsEmailVerified *bool `json:"is_email_verified"`

	FirstName string `json:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" validate:"max=100"`

	IsActive *bool `json:"is_active"`

	// Password is required when creating a user. When updating one it is
	// optional: left empty, the password the user has is kept.
	Password        string `json:"password" validate:"max=128"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`
	// IsPasswordTemporary forces the user to replace the password. Left out:
	// false for a new user, unchanged otherwise.
	IsPasswordTemporary *bool `json:"is_password_temporary"`

	Data map[string]any `json:"data"`
}

func (r *userRequest) clean() {
	r.Email = trimmedLower(r.Email)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
}

func listQuery(c *gin.Context) store.UserQuery {
	q := store.UserQuery{
		Search:   c.Query("search"),
		Limit:    query.Int(c, "limit", 50, maxPageSize),
		Offset:   query.Int(c, "offset", 0, query.MaxOffset),
		Verified: query.Bool(c, "verified"),
	}

	if role, err := uuid.Parse(c.Query("role")); err == nil {
		q.Role = &role
	}

	return q
}

func trimmedLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
