package model

import "time"

type AdminSecurity struct {
	Base

	RequireMFA bool `gorm:"not null" json:"require_mfa"`

	// IdleTimeoutMinutes ends an administrator's session that has not been
	// used for that long, well inside its absolute SessionLifetime.
	IdleTimeoutMinutes int `gorm:"not null;default:30" json:"idle_timeout_minutes"`
}

// The bounds of IdleTimeoutMinutes: under five minutes signs people out
// mid-task, and past the absolute session lifetime it would never apply.
const (
	DefaultAdminIdleTimeout = 30
	MinAdminIdleTimeout     = 5
	MaxAdminIdleTimeout     = 12 * 60
)

func (AdminSecurity) TableName() string {
	return "admin_security"
}

// DefaultAdminSecurity is what is read before the settings are seeded: a
// second factor required, as a fresh installation is seeded with unless
// VARD_ADMIN_MFA says otherwise.
func DefaultAdminSecurity() AdminSecurity {
	return AdminSecurity{RequireMFA: true, IdleTimeoutMinutes: DefaultAdminIdleTimeout}
}

// IdleTimeout is IdleTimeoutMinutes as a duration, the default for a row
// that has none.
func (s AdminSecurity) IdleTimeout() time.Duration {
	if s.IdleTimeoutMinutes <= 0 {
		return DefaultAdminIdleTimeout * time.Minute
	}
	return time.Duration(s.IdleTimeoutMinutes) * time.Minute
}
