package otp

import (
	"loginer/internal/model"
)

// settingsResponse is the settings as the panel sees them, built by hand so
// the row's own bookkeeping stays out of a record of settings.
type settingsResponse struct {
	CodeLength      int `json:"code_length"`
	LifetimeMinutes int `json:"lifetime_minutes"`
	MaxAttempts     int `json:"max_attempts"`
	ResendSeconds   int `json:"resend_seconds"`
}

// usingFlow is a login flow that asks for an emailed code.
type usingFlow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	IsEnabled bool   `json:"is_enabled"`
	IsDefault bool   `json:"is_default"`
}

// limits are the bounds the panel holds its fields to, from the model, so the
// two cannot disagree about what a code may be.
type limits struct {
	MinLength       int `json:"min_length"`
	MaxLength       int `json:"max_length"`
	MaxLifetime     int `json:"max_lifetime_minutes"`
	MaxAttempts     int `json:"max_attempts"`
	MaxResendSecond int `json:"max_resend_seconds"`
}

type response struct {
	OTP    settingsResponse `json:"otp"`
	Limits limits           `json:"limits"`
	Flows  []usingFlow      `json:"flows"`
}

func newResponse(settings model.OTPSettings, flows []model.LoginFlow) response {
	using := []usingFlow{}
	for _, flow := range flows {
		if !flow.Offers(model.StepEmailCode) {
			continue
		}

		using = append(using, usingFlow{
			ID:        flow.ID.String(),
			Name:      flow.Name,
			Slug:      flow.Slug,
			IsEnabled: flow.IsEnabled,
			IsDefault: flow.IsDefault,
		})
	}

	return response{
		OTP: settingsResponse{
			CodeLength:      settings.CodeLength,
			LifetimeMinutes: settings.LifetimeMinutes,
			MaxAttempts:     settings.MaxAttempts,
			ResendSeconds:   settings.ResendSeconds,
		},
		Limits: limits{
			MinLength:       model.MinOTPLength,
			MaxLength:       model.MaxOTPLength,
			MaxLifetime:     model.MaxOTPLifetimeMinutes,
			MaxAttempts:     model.MaxOTPAttempts,
			MaxResendSecond: model.MaxOTPResendSeconds,
		},
		Flows: using,
	}
}
