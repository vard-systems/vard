// Package model is the database schema: one struct per table, with the rules
// each row must keep.
package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Base carries the columns every table has. Nothing is soft-deleted: a delete
// removes the row, so what an administrator removes is gone from the database.
type Base struct {
	ID        uuid.UUID `gorm:"type:uuid;primarykey" json:"id"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
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
		&ServerSecret{},
		&OrganizationLogo{},
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
		&APIIntrospector{},
		&UserRole{},
		&User{},
		&SigningKey{},
		&AuthorizationRequest{},
		&PushedRequest{},
		&DeviceAuthorization{},
		&AuthorizationCode{},
		&RefreshToken{},
		&TokenRevocation{},
		&SignInThrottle{},
		&UserSession{},
		&SessionApplication{},
		&FrontchannelLogout{},
		&Consent{},
		&PasswordReset{},
		&EmailVerification{},
		&LoginCode{},
		&UserFactor{},
		&UserRecoveryCode{},
		&PendingSignIn{},
		&PasskeyLogin{},
		&SocialIdentity{},
		&SocialLogin{},
		&SSOConnection{},
		&SSOIdentity{},
		&SSODomainVerification{},
		&SSOLogin{},
		&SAMLAssertion{},
		&DPoPProof{},
	}
}
