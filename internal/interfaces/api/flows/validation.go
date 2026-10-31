package flows

import (
	"net/http"
	"strings"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/validate"
)

var (
	slugTaken    = respond.Define(http.StatusConflict, "login_flow_slug_taken", respond.Admin)
	keepADefault = respond.Define(http.StatusBadRequest, "login_flow_default_required", respond.Admin)
	defaultStays = respond.Define(http.StatusBadRequest, "login_flow_default_protected", respond.Admin)
	flowAbsent   = respond.Define(http.StatusNotFound, "login_flow_not_found", respond.Admin)
)

func (r *flowRequest) applyTo(flow *model.LoginFlow, creating bool) error {
	if creating {
		flow.Slug = slugify(r.Slug, r.Name)
	}

	flow.Name = validate.Text(r.Name, flow.Name)
	flow.Description = validate.Text(r.Description, flow.Description)

	flow.IsDefault = validate.Flag(r.IsDefault, flow.IsDefault)
	flow.IsEnabled = validate.Flag(r.IsEnabled, flow.IsEnabled)

	flow.AllowSignIn = validate.Flag(r.AllowSignIn, flow.AllowSignIn)
	flow.AllowRegistration = validate.Flag(r.AllowRegistration, flow.AllowRegistration)
	flow.AllowPasswordReset = validate.Flag(r.AllowPasswordReset, flow.AllowPasswordReset)
	flow.AllowRememberMe = validate.Flag(r.AllowRememberMe, flow.AllowRememberMe)
	flow.VerifyEmailOnRegister = validate.Flag(r.VerifyEmailOnRegister, flow.VerifyEmailOnRegister)
	flow.RequireVerifiedEmail = validate.Flag(r.RequireVerifiedEmail, flow.RequireVerifiedEmail)
	flow.RequireSecondFactor = validate.Flag(r.RequireSecondFactor, flow.RequireSecondFactor)
	flow.AllowEmailChange = validate.Flag(r.AllowEmailChange, flow.AllowEmailChange)

	if r.SessionLifetimeHours != nil {
		flow.SessionLifetimeHours = *r.SessionLifetimeHours
	}

	if r.Steps != nil {
		flow.Steps = cleanSteps(*r.Steps)
	}

	if flow.IsDefault {
		flow.IsEnabled = true
	}

	if err := flow.Validate(); err != nil {
		return respond.Invalid(err.Error())
	}

	return nil
}

func cleanSteps(sent []model.LoginStep) model.StepList {
	steps := make(model.StepList, 0, len(sent))

	for _, step := range sent {
		tidied := model.LoginStep(strings.ToLower(strings.TrimSpace(string(step))))
		if tidied != "" {
			steps = append(steps, tidied)
		}
	}

	return steps
}

func slugify(sent, name *string) string {
	if sent != nil && strings.TrimSpace(*sent) != "" {
		return strings.ToLower(strings.TrimSpace(*sent))
	}

	if name == nil {
		return ""
	}

	var b strings.Builder
	dash := false

	for _, r := range strings.ToLower(strings.TrimSpace(*name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		case b.Len() > 0 && !dash:
			b.WriteRune('-')
			dash = true
		}
	}

	return strings.TrimSuffix(b.String(), "-")
}
