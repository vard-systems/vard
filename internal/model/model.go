package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base carries the columns every table has. Nothing soft-deletes, so DeletedAt
// is always null and unindexed.
type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-"`
}

func (b *Base) BeforeCreate(*gorm.DB) error {
	if b.ID != uuid.Nil {
		return nil
	}

	id, err := newID()
	if err != nil {
		return err
	}
	b.ID = id

	return nil
}

// newID returns a version 7 UUID: random, but with a timestamp in the leading
// bits, so ids sort by creation time and index writes stay local.
func newID() (uuid.UUID, error) {
	return uuid.NewV7()
}

func All() []any {
	return []any{
		&Organization{},
		&AdminSecurity{},
		&MailSettings{},
		&OTPSettings{},
		&LoginFlow{},
		&Language{},
		&Translation{},
		&SocialProvider{},
		&AdminRole{},
		&Application{},
		&Admin{},
		&AdminRoleAssignment{},
		&AdminSession{},
		&MFA{},
		&AuditLog{},
		&UserField{},
		&API{},
		&APIScope{},
		&ApplicationAPI{},
		&ApplicationAPIScope{},
		&UserRole{},
		&User{},
		&SigningKey{},
		&AuthorizationRequest{},
		&AuthorizationCode{},
		&RefreshToken{},
		&UserSession{},
		&PasswordReset{},
		&EmailVerification{},
		&LoginCode{},
		&SocialIdentity{},
		&SocialLogin{},
		&SSOConnection{},
		&SSOIdentity{},
		&SSOLogin{},
	}
}
