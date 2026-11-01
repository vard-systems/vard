package adminauth

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
)

// adminResponse is built by hand so new columns are never published by
// accident.
type adminResponse struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	AvatarURL string     `json:"avatar_url"`
	Status    string     `json:"status"`
	LastLogin *time.Time `json:"last_login_at,omitempty"`

	// Roles are the names of the roles held for the whole panel.
	Roles []string `json:"roles"`

	// Permissions, per-application ScopedPermissions and IsSuperAdmin drive
	// what the panel shows; the server checks them regardless.
	Permissions       []string               `json:"permissions"`
	ScopedPermissions map[uuid.UUID][]string `json:"scoped_permissions"`
	IsSuperAdmin      bool                   `json:"is_super_admin"`

	// MFAEnabled says whether they sign in with a second factor.
	MFAEnabled bool `json:"mfa_enabled"`
}

func newAdminResponse(a *model.Admin) adminResponse {
	roles := []string{}
	for _, assignment := range a.Assignments {
		if assignment.Global() {
			roles = append(roles, assignment.Role.Name)
		}
	}

	return adminResponse{
		ID:        a.ID.String(),
		Username:  a.Username,
		Email:     a.Email,
		FullName:  a.FullName(),
		FirstName: a.FirstName,
		LastName:  a.LastName,
		AvatarURL: a.AvatarURL,
		Status:    string(a.Status),
		Roles:     roles,
		LastLogin: a.LastLoginAt,

		Permissions:       a.Permissions(),
		ScopedPermissions: a.ScopedPermissions(),
		IsSuperAdmin:      a.IsSuperAdmin(),
		MFAEnabled:        a.HasMFA(),
	}
}

// sessionResponse is one of the places an administrator is signed in.
type sessionResponse struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
	// Current is the session this request came from: the browser reading
	// the list, which is signed out rather than ended from it.
	Current bool `json:"current"`
}

func newSessionResponses(sessions []model.AdminSession, current uuid.UUID) []sessionResponse {
	now := time.Now()
	out := make([]sessionResponse, 0, len(sessions))

	for _, s := range sessions {
		out = append(out, sessionResponse{
			ID:        s.ID.String(),
			IP:        s.IP,
			UserAgent: s.UserAgent,
			CreatedAt: s.CreatedAt,
			ExpiresAt: s.ExpiresAt,
			Active:    s.IsActive(now),
			Current:   s.ID == current,
		})
	}

	return out
}

// organizationBrand is the header's name and app logo, already public on the
// sign-in pages; the logo is named by its version, empty while there is none.
type organizationBrand struct {
	Name        string `json:"name"`
	LogoVersion string `json:"logo_version"`
}

func newOrganizationBrand(organization model.Organization) organizationBrand {
	return organizationBrand{Name: organization.Name, LogoVersion: organization.LogoVersion}
}
