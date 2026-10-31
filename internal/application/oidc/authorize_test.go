package oidc

import (
	"testing"
	"time"

	"vard/internal/domain/model"
)

// A session made by one flow is accepted at /oauth2/authorize for an
// application on another only when it would have been made by that one too.
func TestSessionSatisfiesTheApplicationsFlow(t *testing.T) {
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

	session := func(verified bool, age time.Duration) *Session {
		return &Session{
			Record: model.UserSession{AuthenticatedAt: now.Add(-age)},
			User:   &model.User{IsEmailVerified: verified},
		}
	}

	tests := []struct {
		name    string
		change  func(*model.LoginFlow)
		session *Session
		want    bool
	}{
		{
			name:    "a fresh session on the default flow",
			change:  func(*model.LoginFlow) {},
			session: session(true, time.Hour),
			want:    true,
		},
		{
			name:    "the default flow takes no unverified address",
			change:  func(*model.LoginFlow) {},
			session: session(false, time.Hour),
			want:    false,
		},
		{
			name:    "a flow with sign-ins closed takes no session",
			change:  func(f *model.LoginFlow) { f.AllowSignIn = false },
			session: session(true, time.Hour),
			want:    false,
		},
		{
			name:    "a flow that requires a verified address, and an unverified one",
			change:  func(f *model.LoginFlow) { f.RequireVerifiedEmail = true },
			session: session(false, time.Hour),
			want:    false,
		},
		{
			name:    "a flow that requires a verified address, and a verified one",
			change:  func(f *model.LoginFlow) { f.RequireVerifiedEmail = true },
			session: session(true, time.Hour),
			want:    true,
		},
		{
			name:    "a session older than the flow lets one live",
			change:  func(f *model.LoginFlow) { f.SessionLifetimeHours = 8 },
			session: session(true, 9*time.Hour),
			want:    false,
		},
		{
			name:    "a session within the flow's lifetime",
			change:  func(f *model.LoginFlow) { f.SessionLifetimeHours = 8 },
			session: session(true, 7*time.Hour),
			want:    true,
		},
		{
			name:    "a flow with an emailed code, which a session cannot show it typed",
			change:  func(f *model.LoginFlow) { f.Steps = append(f.Steps, model.StepEmailCode) },
			session: session(true, time.Minute),
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flow := model.DefaultLoginFlow()
			tt.change(&flow)

			if got := sessionSatisfies(&flow, tt.session, now); got != tt.want {
				t.Errorf("sessionSatisfies = %v, want %v", got, tt.want)
			}
		})
	}
}
