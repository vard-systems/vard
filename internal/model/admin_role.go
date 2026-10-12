package model

const RoleSuperAdmin = "super_admin"

type AdminRole struct {
	Base

	Name        string `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"`
	Description string `gorm:"size:255" json:"description,omitempty"`

	Permissions []string `gorm:"type:text;serializer:json" json:"permissions"`
}

func (AdminRole) TableName() string {
	return "admin_roles"
}

func (r AdminRole) IsSuperAdmin() bool {
	return r.Name == RoleSuperAdmin
}

func (r AdminRole) Grants(permission string) bool {
	if r.IsSuperAdmin() {
		return true
	}

	for _, granted := range r.Permissions {
		if granted == permission {
			return true
		}
	}

	return false
}
