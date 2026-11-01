package flows

import "vard/internal/domain/model"

// targetType is what a flow is called in the activity log.
const targetType = "login_flow"

// flowRequest is a PATCH: nil leaves a setting alone. On create, omitted
// settings come from the default flow.
type flowRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`

	IsDefault *bool `json:"is_default"`
	IsEnabled *bool `json:"is_enabled"`

	Steps *[]model.LoginStep `json:"steps"`

	AllowSignIn           *bool `json:"allow_sign_in"`
	AllowRegistration     *bool `json:"allow_registration"`
	AllowPasswordReset    *bool `json:"allow_password_reset"`
	AllowRememberMe       *bool `json:"allow_remember_me"`
	VerifyEmailOnRegister *bool `json:"verify_email_on_register"`
	RequireVerifiedEmail  *bool `json:"require_verified_email"`
	RequireSecondFactor   *bool `json:"require_second_factor"`
	AllowEmailChange      *bool `json:"allow_email_change"`
	SessionLifetimeHours  *int  `json:"session_lifetime_hours"`
}
