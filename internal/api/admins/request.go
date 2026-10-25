package admins

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/internal/api/kit/query"
	"vard/internal/model"
	"vard/internal/store"
)

const targetType = "admin_user"

const maxPageSize = 200

// adminRequest is the create and update body. The address is the username, and
// assignments replace every role held.
type adminRequest struct {
	Email     string `json:"email" validate:"required,email,max=255"`
	FirstName string `json:"first_name" validate:"required,max=100"`
	LastName  string `json:"last_name" validate:"max=100"`

	// Status says whether the account may sign in. Only an active one may.
	Status string `json:"status" validate:"required,oneof=active suspended disabled"`

	// Password is required when creating an administrator. When updating one
	// it is optional: left empty, the password they have is kept.
	Password        string `json:"password" validate:"max=72"`
	ConfirmPassword string `json:"confirm_password" validate:"eqfield=Password"`

	Assignments []assignmentRequest `json:"assignments"`
}

// assignmentRequest gives the administrator one role, for the whole panel
// when ApplicationID is null or for that one application.
type assignmentRequest struct {
	RoleID        uuid.UUID  `json:"role_id"`
	ApplicationID *uuid.UUID `json:"application_id"`
}

func (r *adminRequest) clean() {
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Status = strings.TrimSpace(r.Status)
}

func listQuery(c *gin.Context) store.AdminQuery {
	q := store.AdminQuery{
		Search: c.Query("search"),
		Limit:  query.Int(c, "limit", 50, maxPageSize),
		Offset: query.Int(c, "offset", 0, query.MaxOffset),
	}

	if status := model.Status(c.Query("status")); status.Valid() {
		q.Status = status
	}

	if role, err := uuid.Parse(c.Query("role")); err == nil {
		q.Role = &role
	}

	return q
}

// securityRequest changes the settings for every administrator; an omitted flag
// is left alone.
type securityRequest struct {
	RequireMFA *bool `json:"require_mfa"`
}
