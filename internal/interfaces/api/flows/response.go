package flows

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/domain/model"
)

// flowResponse is a flow plus how many applications use it and whether all its
// steps are implemented.
type flowResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`

	IsDefault bool `json:"is_default"`
	IsEnabled bool `json:"is_enabled"`

	Steps []model.LoginStep `json:"steps"`

	AllowSignIn           bool `json:"allow_sign_in"`
	AllowRegistration     bool `json:"allow_registration"`
	AllowPasswordReset    bool `json:"allow_password_reset"`
	AllowRememberMe       bool `json:"allow_remember_me"`
	VerifyEmailOnRegister bool `json:"verify_email_on_register"`
	RequireVerifiedEmail  bool `json:"require_verified_email"`
	AllowEmailChange      bool `json:"allow_email_change"`
	SessionLifetimeHours  int  `json:"session_lifetime_hours"`

	// Applications is how many applications name this flow. The ones that
	// name none are not counted: they follow whichever flow is the default.
	Applications int `json:"applications"`

	// Planned are the steps this flow names that the sign-in pages do not run
	// yet. Empty means the flow is the sign-in it describes.
	Planned []model.LoginStep `json:"planned"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// listResponse is what the list endpoint answers with: the flows, and the
// catalog the panel builds its step picker from.
type listResponse struct {
	Flows []flowResponse        `json:"flows"`
	Steps []model.LoginStepSpec `json:"step_kinds"`
}

type response struct {
	Flow flowResponse `json:"flow"`
}

func newListResponse(flows []model.LoginFlow, counts map[uuid.UUID]int) listResponse {
	out := listResponse{
		Flows: make([]flowResponse, 0, len(flows)),
		Steps: model.LoginStepSpecs,
	}

	for _, flow := range flows {
		out.Flows = append(out.Flows, newFlowResponse(flow, counts[flow.ID]))
	}

	return out
}

func newFlowResponse(flow model.LoginFlow, applications int) flowResponse {
	return flowResponse{
		ID:                    flow.ID,
		Name:                  flow.Name,
		Slug:                  flow.Slug,
		Description:           flow.Description,
		IsDefault:             flow.IsDefault,
		IsEnabled:             flow.IsEnabled,
		Steps:                 append([]model.LoginStep(nil), flow.Steps...),
		AllowSignIn:           flow.AllowSignIn,
		AllowRegistration:     flow.AllowRegistration,
		AllowPasswordReset:    flow.AllowPasswordReset,
		AllowRememberMe:       flow.AllowRememberMe,
		VerifyEmailOnRegister: flow.VerifyEmailOnRegister,
		RequireVerifiedEmail:  flow.RequireVerifiedEmail,
		AllowEmailChange:      flow.AllowEmailChange,
		SessionLifetimeHours:  flow.SessionLifetimeHours,
		Applications:          applications,
		Planned:               plannedSteps(flow),
		CreatedAt:             flow.CreatedAt,
		UpdatedAt:             flow.UpdatedAt,
	}
}

func plannedSteps(flow model.LoginFlow) []model.LoginStep {
	planned := make([]model.LoginStep, 0)

	for _, step := range flow.Steps {
		if spec, known := model.LoginStepSpecOf(step); known && !spec.Implemented {
			planned = append(planned, step)
		}
	}

	return planned
}
