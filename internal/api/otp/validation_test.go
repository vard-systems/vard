package otp

import (
	"errors"
	"reflect"
	"testing"

	"loginer/internal/api/kit/respond"
	"loginer/internal/model"
)

func TestSettingsRequestApplyTo(t *testing.T) {
	tests := []struct {
		name    string
		request settingsRequest
		want    string   // the message, or "" when the request is fine
		changed []string // the settings it should report as changed
	}{
		{
			name:    "nothing sent changes nothing",
			request: settingsRequest{},
			changed: []string{},
		},
		{
			name:    "a longer code",
			request: settingsRequest{CodeLength: new(8)},
			changed: []string{"code_length"},
		},
		{
			name:    "the same values again are no change",
			request: settingsRequest{CodeLength: new(6), MaxAttempts: new(5)},
			changed: []string{},
		},
		{
			name: "every setting at once, in the order they are listed",
			request: settingsRequest{
				CodeLength:      new(4),
				LifetimeMinutes: new(5),
				MaxAttempts:     new(3),
				ResendSeconds:   new(30),
			},
			changed: []string{"code_length", "lifetime_minutes", "max_attempts", "resend_seconds"},
		},
		{
			name:    "no wait between messages",
			request: settingsRequest{ResendSeconds: new(0)},
			changed: []string{"resend_seconds"},
		},
		{
			name:    "a code too short to be worth guessing at",
			request: settingsRequest{CodeLength: new(3)},
			want:    "code_length must be between 4 and 10",
		},
		{
			name:    "a code that outlives the message it came in",
			request: settingsRequest{LifetimeMinutes: new(1440)},
			want:    "lifetime_minutes must be between 1 and 60",
		},
		{
			name:    "guesses without end",
			request: settingsRequest{MaxAttempts: new(0)},
			want:    "max_attempts must be between 1 and 10",
		},
		{
			name:    "a wait that is a wall",
			request: settingsRequest{ResendSeconds: new(3600)},
			want:    "resend_seconds must be between 0 and 900",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			settings := model.DefaultOTPSettings()

			changed, err := request.applyTo(&settings)

			if tt.want != "" {
				var fault respond.Fault
				if !errors.As(err, &fault) {
					t.Fatalf("applyTo() = %v, want a Fault", err)
				}
				if fault.Message != tt.want {
					t.Errorf("message = %q, want %q", fault.Message, tt.want)
				}
				// Nothing may be written when a request is refused.
				if !reflect.DeepEqual(settings, model.DefaultOTPSettings()) {
					t.Errorf("the settings were changed by a refused request")
				}
				return
			}

			if err != nil {
				t.Fatalf("applyTo() = %v, want nothing", err)
			}
			if !reflect.DeepEqual(changed, tt.changed) {
				t.Errorf("changed = %v, want %v", changed, tt.changed)
			}
		})
	}
}

// The page is told what a code may be, so its fields and the model cannot
// disagree about what will be refused.
func TestResponseCarriesTheModelsLimits(t *testing.T) {
	answer := newResponse(model.DefaultOTPSettings(), nil)

	for _, check := range []struct {
		name string
		got  int
		want int
	}{
		{"min_length", answer.Limits.MinLength, model.MinOTPLength},
		{"max_length", answer.Limits.MaxLength, model.MaxOTPLength},
		{"max_lifetime_minutes", answer.Limits.MaxLifetime, model.MaxOTPLifetimeMinutes},
		{"max_attempts", answer.Limits.MaxAttempts, model.MaxOTPAttempts},
		{"max_resend_seconds", answer.Limits.MaxResendSecond, model.MaxOTPResendSeconds},
	} {
		if check.got != check.want {
			t.Errorf("%s = %d, want %d", check.name, check.got, check.want)
		}
	}
}

// Only the flows that ask for a code are listed: the page says who these
// settings are for, and a flow that does not use them is not that.
func TestResponseListsOnlyTheFlowsThatAskForACode(t *testing.T) {
	flows := []model.LoginFlow{
		{Name: "Default", Slug: "default", Steps: model.StepList{model.StepIdentifier, model.StepPassword}},
		{Name: "Staff", Slug: "staff", Steps: model.StepList{model.StepIdentifier, model.StepPassword, model.StepEmailCode}},
	}

	answer := newResponse(model.DefaultOTPSettings(), flows)

	if len(answer.Flows) != 1 {
		t.Fatalf("flows = %d, want 1", len(answer.Flows))
	}
	if answer.Flows[0].Slug != "staff" {
		t.Errorf("flow = %s, want staff", answer.Flows[0].Slug)
	}
}
