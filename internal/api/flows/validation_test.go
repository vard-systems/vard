package flows

import (
	"errors"
	"reflect"
	"testing"

	"vard/internal/api/kit/respond"
	"vard/internal/model"
)

func TestFlowRequestApplyToCreating(t *testing.T) {
	tests := []struct {
		name    string
		request flowRequest
		want    string // the message, or "" when the request is fine
		check   func(model.LoginFlow) error
	}{
		{
			name:    "a flow named and nothing else",
			request: flowRequest{Name: new("  Staff sign-in  ")},
			check: func(f model.LoginFlow) error {
				switch {
				case f.Name != "Staff sign-in":
					return errors.New("name = " + f.Name)
				case f.Slug != "staff-sign-in":
					return errors.New("slug = " + f.Slug + ", want one made from the name")
				case f.IsDefault:
					return errors.New("a new flow made itself the default")
				case f.IsEnabled:
					return errors.New("a new flow offered itself before anyone turned it on")
				}
				return nil
			},
		},
		{
			name:    "an identifier of its own",
			request: flowRequest{Name: new("Staff sign-in"), Slug: new("  STAFF  ")},
			check: func(f model.LoginFlow) error {
				if f.Slug != "staff" {
					return errors.New("slug = " + f.Slug)
				}
				return nil
			},
		},
		{
			name: "steps with the blanks a form sends",
			request: flowRequest{
				Name:  new("Codes and a password"),
				Steps: new([]model.LoginStep{"Identifier", "", " email_code ", "password"}),
			},
			check: func(f model.LoginFlow) error {
				want := model.StepList{model.StepIdentifier, model.StepEmailCode, model.StepPassword}
				if !reflect.DeepEqual(f.Steps, want) {
					return errors.New("the steps were not tidied")
				}
				return nil
			},
		},
		{
			name:    "making it the default also turns it on",
			request: flowRequest{Name: new("Staff"), IsDefault: new(true), IsEnabled: new(false)},
			check: func(f model.LoginFlow) error {
				if !f.IsEnabled {
					return errors.New("the default flow was left turned off")
				}
				return nil
			},
		},
		{
			name:    "no name",
			request: flowRequest{Slug: new("staff")},
			want:    "name is required",
		},
		{
			name:    "a flow that asks for nothing",
			request: flowRequest{Name: new("Open door"), Steps: new([]model.LoginStep{"identifier"})},
			want:    "a flow must ask for at least one of: password, social",
		},
		{
			name:    "a session that lasts a year",
			request: flowRequest{Name: new("Staff"), SessionLifetimeHours: new(24 * 365)},
			want:    "session_lifetime_hours must be between 1 and 2160",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request

			flow := model.DefaultLoginFlow()
			flow.IsDefault, flow.IsEnabled = false, false
			flow.Name, flow.Slug, flow.Description = "", "", ""

			err := request.applyTo(&flow, true)

			if tt.want != "" {
				var fault respond.Fault
				if !errors.As(err, &fault) {
					t.Fatalf("applyTo() = %v, want a Fault", err)
				}
				if fault.Message != tt.want {
					t.Errorf("message = %q, want %q", fault.Message, tt.want)
				}
				return
			}

			if err != nil {
				t.Fatalf("applyTo() = %v, want nothing", err)
			}
			if tt.check != nil {
				if err := tt.check(flow); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// An update says what it changes and nothing else: turning a flow off must
// not empty the steps it was built out of.
func TestFlowRequestUpdatesOnlyWhatItMentions(t *testing.T) {
	flow := model.DefaultLoginFlow()
	flow.IsDefault = false
	was := flow

	request := flowRequest{IsEnabled: new(false)}

	if err := request.applyTo(&flow, false); err != nil {
		t.Fatalf("applyTo() = %v, want nothing", err)
	}

	switch {
	case flow.IsEnabled:
		t.Error("the flow was not turned off")
	case flow.Name != was.Name || flow.Description != was.Description:
		t.Errorf("the name or the description was emptied: %+v", flow)
	case !reflect.DeepEqual(flow.Steps, was.Steps):
		t.Errorf("the steps were emptied: %v", flow.Steps)
	case flow.SessionLifetimeHours != was.SessionLifetimeHours:
		t.Error("a setting nobody mentioned was changed")
	case !flow.AllowRegistration || !flow.AllowPasswordReset:
		t.Error("a switch nobody mentioned was thrown")
	case !flow.AllowSignIn || !flow.AllowRememberMe:
		t.Error("a switch nobody mentioned was thrown")
	}
}

// Every switch a flow has can be set on its own, and setting one leaves the
// rest where they were. They decide who gets in, so a request that mentions
// one must not carry the others with it.
func TestFlowRequestSwitchesAreIndependent(t *testing.T) {
	switches := []struct {
		name string
		set  func(*flowRequest, bool)
		read func(model.LoginFlow) bool
	}{
		{"allow_sign_in", func(r *flowRequest, v bool) { r.AllowSignIn = new(v) }, func(f model.LoginFlow) bool { return f.AllowSignIn }},
		{"allow_registration", func(r *flowRequest, v bool) { r.AllowRegistration = new(v) }, func(f model.LoginFlow) bool { return f.AllowRegistration }},
		{"allow_password_reset", func(r *flowRequest, v bool) { r.AllowPasswordReset = new(v) }, func(f model.LoginFlow) bool { return f.AllowPasswordReset }},
		{"allow_remember_me", func(r *flowRequest, v bool) { r.AllowRememberMe = new(v) }, func(f model.LoginFlow) bool { return f.AllowRememberMe }},
		{"verify_email_on_register", func(r *flowRequest, v bool) { r.VerifyEmailOnRegister = new(v) }, func(f model.LoginFlow) bool { return f.VerifyEmailOnRegister }},
		{"require_verified_email", func(r *flowRequest, v bool) { r.RequireVerifiedEmail = new(v) }, func(f model.LoginFlow) bool { return f.RequireVerifiedEmail }},
		{"allow_email_change", func(r *flowRequest, v bool) { r.AllowEmailChange = new(v) }, func(f model.LoginFlow) bool { return f.AllowEmailChange }},
	}

	for _, one := range switches {
		for _, want := range []bool{true, false} {
			t.Run(one.name, func(t *testing.T) {
				flow := model.DefaultLoginFlow()

				var request flowRequest
				one.set(&request, want)

				if err := request.applyTo(&flow, false); err != nil {
					t.Fatalf("applyTo() = %v", err)
				}

				if got := one.read(flow); got != want {
					t.Errorf("%s = %v, want %v", one.name, got, want)
				}

				// Nothing else moved: every other switch is still the
				// default's, whatever this one was set to.
				def := model.DefaultLoginFlow()
				for _, other := range switches {
					if other.name == one.name {
						continue
					}
					if other.read(flow) != other.read(def) {
						t.Errorf("setting %s also moved %s", one.name, other.name)
					}
				}
			})
		}
	}
}

// The identifier is what an application stores to name its flow, so an update
// may not move it.
func TestFlowRequestKeepsTheSlug(t *testing.T) {
	flow := model.DefaultLoginFlow()

	request := flowRequest{Slug: new("something-else"), Name: new("Renamed")}
	if err := request.applyTo(&flow, false); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	switch {
	case flow.Slug != "default":
		t.Errorf("an update changed the identifier: %s", flow.Slug)
	case flow.Name != "Renamed":
		t.Errorf("the name did not change: %s", flow.Name)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		from string
		want string
	}{
		{name: "a name with spaces", from: "Staff sign-in", want: "staff-sign-in"},
		{name: "a name with punctuation", from: "Staff (2024)!", want: "staff-2024"},
		{name: "a name that is already one", from: "staff", want: "staff"},
		{name: "a name of nothing but punctuation", from: "!!!", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := slugify(nil, &tt.from); got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.from, got, tt.want)
			}
		})
	}
}
