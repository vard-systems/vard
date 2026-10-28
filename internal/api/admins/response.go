package admins

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/model"
	"vard/internal/store"
)

// reference names another row: enough to show it and to send it back.
type reference struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// assignmentResponse is one role an administrator holds, and where.
type assignmentResponse struct {
	ID   uuid.UUID `json:"id"`
	Role reference `json:"role"`
	// Application is the application the role is held for, or null for the
	// whole panel.
	Application *reference `json:"application"`
}

// adminResponse is an administrator as the panel sees it. It is built by hand
// so the password hash and the lockout counters are never published.
type adminResponse struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	FullName  string    `json:"full_name"`
	Status    string    `json:"status"`

	Assignments []assignmentResponse `json:"assignments"`
	// Permissions come from panel-wide roles, ScopedPermissions from roles held
	// per application, and IsSuperAdmin from holding super_admin panel-wide.
	Permissions       []string               `json:"permissions"`
	ScopedPermissions map[uuid.UUID][]string `json:"scoped_permissions"`
	IsSuperAdmin      bool                   `json:"is_super_admin"`

	// MFAEnabled says whether they sign in with a second factor.
	MFAEnabled bool `json:"mfa_enabled"`

	LastLoginAt *time.Time `json:"last_login_at"`
	LastLoginIP string     `json:"last_login_ip"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func newAdminResponse(admin model.Admin) adminResponse {
	assignments := make([]assignmentResponse, 0, len(admin.Assignments))
	for _, a := range admin.Assignments {
		out := assignmentResponse{ID: a.ID, Role: reference{ID: a.Role.ID, Name: a.Role.Name}}
		if a.ApplicationID != nil {
			out.Application = &reference{ID: *a.ApplicationID}
			if a.Application != nil {
				out.Application.Name = a.Application.Name
			}
		}
		assignments = append(assignments, out)
	}

	return adminResponse{
		ID:                admin.ID,
		Username:          admin.Username,
		Email:             admin.Email,
		FirstName:         admin.FirstName,
		LastName:          admin.LastName,
		FullName:          admin.FullName(),
		Status:            string(admin.Status),
		Assignments:       assignments,
		Permissions:       admin.Permissions(),
		ScopedPermissions: admin.ScopedPermissions(),
		IsSuperAdmin:      admin.IsSuperAdmin(),
		MFAEnabled:        admin.HasMFA(),
		LastLoginAt:       admin.LastLoginAt,
		LastLoginIP:       admin.LastLoginIP,
		CreatedAt:         admin.CreatedAt,
		UpdatedAt:         admin.UpdatedAt,
	}
}

type pageResponse struct {
	Admins []adminResponse `json:"admins"`
	Total  int64           `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

func newPageResponse(admins []model.Admin, total int64, query store.AdminQuery) pageResponse {
	out := make([]adminResponse, 0, len(admins))
	for _, admin := range admins {
		out = append(out, newAdminResponse(admin))
	}

	return pageResponse{Admins: out, Total: total, Limit: query.Limit, Offset: query.Offset}
}

// securityResponse is the administrators' sign-in settings and how many would
// have to enrol if a second factor were required.
type securityResponse struct {
	RequireMFA         bool `json:"require_mfa"`
	IdleTimeoutMinutes int  `json:"idle_timeout_minutes"`
	// Administrators is how many there are, and WithMFA how many of those
	// already sign in with a second factor.
	Administrators int64 `json:"administrators"`
	WithMFA        int64 `json:"with_mfa"`
}

func newSecurityResponse(security model.AdminSecurity, total, withMFA int64) securityResponse {
	return securityResponse{
		RequireMFA:         security.RequireMFA,
		IdleTimeoutMinutes: int(security.IdleTimeout().Minutes()),
		Administrators:     total,
		WithMFA:            withMFA,
	}
}
