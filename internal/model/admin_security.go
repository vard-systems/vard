package model

type AdminSecurity struct {
	Base

	RequireMFA bool `gorm:"not null" json:"require_mfa"`
}

func (AdminSecurity) TableName() string {
	return "admin_security"
}

// DefaultAdminSecurity makes a second factor optional, so the first
// administrator is not forced to enrol on a server they are only trying out.
// Requiring it is a deliberate choice on the Administrators page.
func DefaultAdminSecurity() AdminSecurity {
	return AdminSecurity{RequireMFA: false}
}
