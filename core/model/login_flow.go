package model

import (
	"fmt"
	"slices"
	"strings"
)

type LoginFlow struct {
	Base

	Name string `gorm:"size:100;not null" json:"name"`

	Slug string `gorm:"size:64;not null;uniqueIndex" json:"slug"`

	Description string `gorm:"size:500;not null;default:''" json:"description"`

	IsDefault bool `gorm:"not null" json:"is_default"`

	IsEnabled bool `gorm:"not null" json:"is_enabled"`

	Steps StepList `gorm:"type:jsonb;serializer:json;not null" json:"steps"`

	AllowSignIn bool `gorm:"not null" json:"allow_sign_in"`

	AllowRegistration bool `gorm:"not null" json:"allow_registration"`

	AllowPasswordReset bool `gorm:"not null" json:"allow_password_reset"`

	AllowRememberMe bool `gorm:"not null" json:"allow_remember_me"`

	VerifyEmailOnRegister bool `gorm:"not null" json:"verify_email_on_register"`

	RequireVerifiedEmail bool `gorm:"not null" json:"require_verified_email"`

	// RequireSecondFactor asks every sign-in to the flow's applications for
	// one of the account's own second factors, not only those of people who
	// turned one on; a session that has not passed one is asked for it.
	RequireSecondFactor bool `gorm:"not null" json:"require_second_factor"`

	AllowEmailChange bool `gorm:"not null" json:"allow_email_change"`

	SessionLifetimeHours int `gorm:"not null" json:"session_lifetime_hours"`
}

func (LoginFlow) TableName() string {
	return "login_flows"
}

type StepList []LoginStep

type LoginStep string

const (
	StepIdentifier LoginStep = "identifier"

	StepPassword LoginStep = "password"

	StepSocial LoginStep = "social"

	StepEmailCode LoginStep = "email_code"

	StepTOTP LoginStep = "totp"

	StepTerms LoginStep = "terms"

	StepConsent LoginStep = "consent"
)

// LoginStepSpec describes one step for the panel: what it is called, what it
// does, and whether this server runs it yet.
type LoginStepSpec struct {
	Step        LoginStep `json:"step"`
	Label       string    `json:"label"`
	Description string    `json:"description"`

	// Fixed steps cannot be removed or moved: a flow that does not ask who
	// is signing in has nobody to sign in.
	Fixed bool `json:"fixed"`

	// Implemented says the sign-in pages run this step today; others can be
	// planned and are marked in the panel.
	Implemented bool `json:"implemented"`
}

var LoginStepSpecs = []LoginStepSpec{
	{
		Step:        StepIdentifier,
		Label:       "Identify",
		Description: "Ask for the email address the account is under.",
		Fixed:       true,
		Implemented: true,
	},
	{
		Step:        StepPassword,
		Label:       "Password",
		Description: "Ask for the account's password.",
		Implemented: true,
	},
	{
		Step:        StepSocial,
		Label:       "Another account",
		Description: "Offer the providers on the Social page beside the first step.",
		Implemented: true,
	},
	{
		Step:        StepEmailCode,
		Label:       "Emailed code",
		Description: "Send a one-time code to the address and ask for it.",
		Implemented: true,
	},
	{
		Step:        StepTOTP,
		Label:       "Two-step verification",
		Description: "An authenticator app or passkeys, asked at every sign-in of anyone who has one. Set up on the account page, or while signing in where required.",
		Implemented: true,
	},
	{
		Step:        StepTerms,
		Label:       "Agreements",
		Description: "Ask for the organisation's terms and privacy policy to be accepted, and again when they change.",
		Implemented: true,
	},
	{
		Step:        StepConsent,
		Label:       "Consent",
		Description: "Show what the application is asking for and wait to be allowed, once for each new thing it asks.",
		Implemented: true,
	},
}

func LoginStepSpecOf(step LoginStep) (LoginStepSpec, bool) {
	for _, spec := range LoginStepSpecs {
		if spec.Step == step {
			return spec, true
		}
	}

	return LoginStepSpec{}, false
}

var proofSteps = []LoginStep{StepPassword, StepSocial}

const maxSessionLifetimeHours = 24 * 90

func DefaultLoginFlow() LoginFlow {
	return LoginFlow{
		Name:               "Default sign-in",
		Slug:               "default",
		Description:        "An email address and a password, with the registered providers beside them.",
		IsDefault:          true,
		IsEnabled:          true,
		Steps:              StepList{StepIdentifier, StepPassword, StepSocial},
		AllowSignIn:        true,
		AllowRegistration:  true,
		AllowPasswordReset: true,
		AllowRememberMe:    true,
		// An unproved address may be somebody else's: it is proved before the
		// account is used. Only seeds a fresh installation's flow and new
		// ones; a stored flow keeps its own setting.
		RequireVerifiedEmail: true,
		SessionLifetimeHours: 24 * 14,
	}
}

func (f LoginFlow) Validate() error {
	switch {
	case strings.TrimSpace(f.Name) == "":
		return fmt.Errorf("name is required")
	case len(f.Name) > 100:
		return fmt.Errorf("name must be at most 100 characters")
	}

	switch {
	case f.Slug == "":
		return fmt.Errorf("slug is required")
	case len(f.Slug) > 64:
		return fmt.Errorf("slug must be at most 64 characters")
	case !slugPattern.MatchString(f.Slug):
		return fmt.Errorf("slug must be lower case letters, numbers and dashes, such as staff-sign-in")
	}

	if len(f.Description) > 500 {
		return fmt.Errorf("description must be at most 500 characters")
	}

	if err := f.validateSteps(); err != nil {
		return err
	}

	// A factor is set up and answered through the two-step step; without
	// it nobody could meet the requirement.
	if f.RequireSecondFactor && !f.Offers(StepTOTP) {
		return fmt.Errorf("require_second_factor needs the %s step", StepTOTP)
	}

	if f.SessionLifetimeHours < 1 || f.SessionLifetimeHours > maxSessionLifetimeHours {
		return fmt.Errorf("session_lifetime_hours must be between 1 and %d", maxSessionLifetimeHours)
	}

	if f.IsDefault && !f.IsEnabled {
		return fmt.Errorf("the default flow cannot be turned off")
	}

	return nil
}

func (f LoginFlow) validateSteps() error {
	if len(f.Steps) == 0 {
		return fmt.Errorf("steps is required")
	}

	if f.Steps[0] != StepIdentifier {
		return fmt.Errorf("the first step must be %s", StepIdentifier)
	}

	seen := make(map[LoginStep]bool, len(f.Steps))
	for _, step := range f.Steps {
		if _, known := LoginStepSpecOf(step); !known {
			return fmt.Errorf("step must be one of: %s", strings.Join(LoginStepNames(), ", "))
		}
		if seen[step] {
			return fmt.Errorf("%s is listed twice", step)
		}
		seen[step] = true
	}

	for _, step := range proofSteps {
		if seen[step] {
			return nil
		}
	}

	return fmt.Errorf("a flow must ask for at least one of: %s", joinSteps(proofSteps))
}

func LoginStepNames() []string {
	names := make([]string, 0, len(LoginStepSpecs))
	for _, spec := range LoginStepSpecs {
		names = append(names, string(spec.Step))
	}

	return names
}

func joinSteps(steps []LoginStep) string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		names = append(names, string(step))
	}

	return strings.Join(names, ", ")
}

// PublicLoginOptions is the part of a flow the sign-in pages are told: what
// they may offer, and nothing about how the server checks any of it.
type PublicLoginOptions struct {
	// Steps are the steps that will actually run, in order; planned steps are
	// left out.
	Steps []LoginStep `json:"steps"`

	// AllowSignIn false is a closed door: the pages say so instead of asking
	// for anything.
	AllowSignIn        bool `json:"allow_sign_in"`
	AllowRegistration  bool `json:"allow_registration"`
	AllowPasswordReset bool `json:"allow_password_reset"`

	// AllowRememberMe puts "stay signed in" beside the password.
	AllowRememberMe bool `json:"allow_remember_me"`

	// AllowEmailChange offers "change" beside the address on the account page.
	AllowEmailChange bool `json:"allow_email_change"`
}

func (f LoginFlow) PublicOptions() PublicLoginOptions {
	steps := make([]LoginStep, 0, len(f.Steps))
	for _, step := range f.Steps {
		if spec, known := LoginStepSpecOf(step); known && spec.Implemented {
			steps = append(steps, step)
		}
	}

	return PublicLoginOptions{
		Steps:              steps,
		AllowSignIn:        f.AllowSignIn,
		AllowRegistration:  f.AllowRegistration,
		AllowPasswordReset: f.AllowPasswordReset,
		AllowRememberMe:    f.AllowRememberMe,
		AllowEmailChange:   f.AllowEmailChange,
	}
}

func (f LoginFlow) Offers(step LoginStep) bool {
	return slices.Contains(f.Steps, step)
}
