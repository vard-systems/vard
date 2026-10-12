package otp

import (
	"net/http"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

func (r *settingsRequest) applyTo(settings *model.OTPSettings) ([]string, error) {
	updated := *settings
	updated.CodeLength = validate.Number(r.CodeLength, settings.CodeLength)
	updated.LifetimeMinutes = validate.Number(r.LifetimeMinutes, settings.LifetimeMinutes)
	updated.MaxAttempts = validate.Number(r.MaxAttempts, settings.MaxAttempts)
	updated.ResendSeconds = validate.Number(r.ResendSeconds, settings.ResendSeconds)

	if err := updated.Validate(); err != nil {
		return nil, respond.Fault{Status: http.StatusBadRequest, Message: err.Error()}
	}

	changed := []string{}
	for _, field := range []struct {
		name   string
		was    int
		became int
	}{
		{"code_length", settings.CodeLength, updated.CodeLength},
		{"lifetime_minutes", settings.LifetimeMinutes, updated.LifetimeMinutes},
		{"max_attempts", settings.MaxAttempts, updated.MaxAttempts},
		{"resend_seconds", settings.ResendSeconds, updated.ResendSeconds},
	} {
		if field.was != field.became {
			changed = append(changed, field.name)
		}
	}

	*settings = updated

	return changed, nil
}
