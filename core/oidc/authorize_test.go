package oidc

import (
	"strings"
	"testing"
	"time"

	"vard/core/model"
)

// A session made by one flow is accepted at /oauth2/authorize for an
// application on another only when it would have been made by that one too;
// when only a second factor is missing, it is asked for in the session.
func TestSessionMeetsTheApplicationsFlow(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	type made struct {
		verified bool
		age      time.Duration // since the session began
		methods  []string
		// steppedUp is how long ago a second factor was passed in it, 0 for
		// never: a step-up renews authenticated_at, never created_at.
		steppedUp time.Duration
	}
	session := func(m made) *Session {
		authenticated := now.Add(-m.age)
		if m.steppedUp > 0 {
			authenticated = now.Add(-m.steppedUp)
		}
		return &Session{
			Record: model.UserSession{
				Base:            model.Base{CreatedAt: now.Add(-m.age)},
				AuthenticatedAt: authenticated,
				Methods:         m.methods,
			},
			User: &model.User{IsEmailVerified: m.verified},
		}
	}

	requiring := func(f *model.LoginFlow) {
		f.Steps = append(f.Steps, model.StepTOTP)
		f.RequireSecondFactor = true
	}
	password := []string{model.AMRPassword}
	factor := []string{model.AMRPassword, model.AMROneTimeCode, model.MethodSecondFactor}

	tests := []struct {
		name    string
		change  func(*model.LoginFlow)
		session made
		want    sessionVerdict
	}{
		{
			name:    "a fresh session on the default flow",
			change:  func(*model.LoginFlow) {},
			session: made{verified: true, age: time.Hour, methods: password},
			want:    sessionSatisfied,
		},
		{
			name:    "the default flow takes no unverified address",
			change:  func(*model.LoginFlow) {},
			session: made{age: time.Hour},
			want:    sessionSignInAgain,
		},
		{
			name:    "a flow with sign-ins closed takes no session",
			change:  func(f *model.LoginFlow) { f.AllowSignIn = false },
			session: made{verified: true, age: time.Hour},
			want:    sessionSignInAgain,
		},
		{
			name:    "a flow that requires a verified address, and an unverified one",
			change:  func(f *model.LoginFlow) { f.RequireVerifiedEmail = true },
			session: made{age: time.Hour},
			want:    sessionSignInAgain,
		},
		{
			name:    "a flow that requires a verified address, and a verified one",
			change:  func(f *model.LoginFlow) { f.RequireVerifiedEmail = true },
			session: made{verified: true, age: time.Hour},
			want:    sessionSatisfied,
		},
		{
			name:    "a session older than the flow lets one live",
			change:  func(f *model.LoginFlow) { f.SessionLifetimeHours = 8 },
			session: made{verified: true, age: 9 * time.Hour},
			want:    sessionSignInAgain,
		},
		{
			name:    "a session within the flow's lifetime",
			change:  func(f *model.LoginFlow) { f.SessionLifetimeHours = 8 },
			session: made{verified: true, age: 7 * time.Hour},
			want:    sessionSatisfied,
		},
		{
			name:    "a flow with an emailed code, which a session cannot show it typed",
			change:  func(f *model.LoginFlow) { f.Steps = append(f.Steps, model.StepEmailCode) },
			session: made{verified: true, age: time.Minute},
			want:    sessionSignInAgain,
		},
		{
			name:    "a flow that requires a second factor, and a session that passed one",
			change:  requiring,
			session: made{verified: true, age: time.Hour, methods: factor},
			want:    sessionSatisfied,
		},
		{
			name:    "a flow that requires a second factor, and a password alone",
			change:  requiring,
			session: made{verified: true, age: time.Hour, methods: password},
			want:    sessionStepUp,
		},
		{
			// Also a session from before sessions recorded the marker: its
			// authenticator code reads the same, so it steps up once.
			name:    "a password and an emailed code are not a second factor",
			change:  requiring,
			session: made{verified: true, age: time.Hour, methods: []string{model.AMRPassword, model.AMROneTimeCode}},
			want:    sessionStepUp,
		},
		{
			name:    "a passkey sign-in",
			change:  requiring,
			session: made{verified: true, age: time.Hour, methods: []string{model.AMRHardwareKey, model.AMRUserPresence, model.MethodSecondFactor}},
			want:    sessionSatisfied,
		},
		{
			name:    "single sign-on, which proved nothing here",
			change:  requiring,
			session: made{verified: true, age: time.Hour},
			want:    sessionStepUp,
		},
		{
			name: "a stepped-up session past the flow's lifetime signs in again",
			change: func(f *model.LoginFlow) {
				requiring(f)
				f.SessionLifetimeHours = 8
			},
			session: made{verified: true, age: 9 * time.Hour, methods: factor, steppedUp: time.Minute},
			want:    sessionSignInAgain,
		},
		{
			name:    "a session past its lifetime is not offered a step-up",
			change:  func(f *model.LoginFlow) { requiring(f); f.SessionLifetimeHours = 8 },
			session: made{verified: true, age: 9 * time.Hour, methods: password},
			want:    sessionSignInAgain,
		},
		{
			name:    "an unverified address is not offered a step-up",
			change:  func(f *model.LoginFlow) { requiring(f); f.RequireVerifiedEmail = true },
			session: made{age: time.Hour, methods: password},
			want:    sessionSignInAgain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := model.DefaultLoginFlow()
			tt.change(&flow)

			if got := sessionMeets(&flow, session(tt.session), now); got != tt.want {
				t.Errorf("sessionMeets = %v, want %v", got, tt.want)
			}
		})
	}
}

// prompt=none never shows a page: each page it would have needed is a
// distinct error for the client.
func TestSilentRefusalNamesWhatWasNeeded(t *testing.T) {
	tests := []struct {
		page string
		code string
		says string
	}{
		{PageConsent, ErrConsentRequired, "has not allowed"},
		{PageAgreements, ErrInteractionRequired, "terms"},
		{PageSecondFactor, ErrInteractionRequired, "second factor"},
	}

	for _, tt := range tests {
		t.Run(tt.page, func(t *testing.T) {
			code, description := silentRefusal(tt.page)
			if code != tt.code {
				t.Errorf("silentRefusal(%q) = %q, want %q", tt.page, code, tt.code)
			}
			if !strings.Contains(description, tt.says) {
				t.Errorf("silentRefusal(%q) says %q, want it to mention %q", tt.page, description, tt.says)
			}
		})
	}
}
