package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type User struct {
	Base

	Email           string `gorm:"uniqueIndex;size:255;not null;check:chk_users_email_lower,email = lower(email)" json:"email"`
	IsEmailVerified bool   `gorm:"not null;default:false" json:"is_email_verified"`
	FirstName       string `gorm:"size:100;not null;default:''" json:"first_name"`
	LastName        string `gorm:"size:100;not null;default:''" json:"last_name"`
	IsActive        bool   `gorm:"not null" json:"is_active"`

	PasswordHash string `gorm:"size:255;not null;default:''" json:"-"`
	// IsPasswordTemporary marks a password an administrator chose for the
	// user, which the user is expected to replace the next time they sign in.
	IsPasswordTemporary bool `gorm:"not null;default:false" json:"is_password_temporary"`
	// HasPassword tells the panel whether a password is set, without telling
	// it anything about the password.
	HasPassword bool `gorm:"-" json:"has_password"`

	// TwoStepEnabled tells the panel the user turned on two-step
	// verification, so an administrator can reset it for someone who lost
	// every factor.
	TwoStepEnabled bool `gorm:"-" json:"two_step_enabled"`

	// SocialAccounts are the providers this user signs in with, loaded from
	// social_identities.
	SocialAccounts []SocialAccount `gorm:"-" json:"social_accounts"`

	// LastLoginAt is when the user last signed in to an application.
	LastLoginAt *time.Time `json:"last_login_at"`

	// AgreementsAcceptedAt is when the user last accepted the organization's
	// terms and privacy policy, on the Agreements step.
	AgreementsAcceptedAt *time.Time `json:"agreements_accepted_at"`

	Data map[string]any `gorm:"type:jsonb;serializer:json" json:"data"`

	// Roles are the roles given to this user directly. What they add up to,
	// inheritance included, is RoleGraph.Effective.
	Roles []UserRole `gorm:"many2many:user_role_members;constraint:OnDelete:CASCADE" json:"roles"`
}

func (User) TableName() string {
	return "users"
}

// NormalizeEmail trims and lower-cases an address, so the unique index on
// users.email both finds users and stops "Ada@x" duplicating "ada@x".
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (u *User) BeforeSave(*gorm.DB) error {
	u.Email = NormalizeEmail(u.Email)
	return nil
}

func (u *User) AfterFind(*gorm.DB) error {
	u.HasPassword = u.PasswordHash != ""
	return nil
}

func (u *User) SetPassword(password string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	u.PasswordHash = hash
	u.HasPassword = true

	return nil
}

func (u User) CanSignIn() bool {
	return u.IsActive
}

func (u User) FullName() string {
	switch {
	case u.FirstName != "" && u.LastName != "":
		return u.FirstName + " " + u.LastName
	case u.FirstName != "":
		return u.FirstName
	case u.LastName != "":
		return u.LastName
	default:
		return u.Email
	}
}
