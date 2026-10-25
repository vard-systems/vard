package oidc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"vard/internal/model"
)

// Changing how someone signs in needs them at the keyboard. Without a
// password to ask for, only a sign-in of the last few minutes will do; an
// application's token never will.
func TestProvePresenceWithoutAPassword(t *testing.T) {
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	s := &Service{now: func() time.Time { return now }}

	tests := []struct {
		name    string
		session Session
		want    error
	}{
		{
			name:    "a browser that signed in just now",
			session: Session{Record: model.UserSession{Base: model.Base{ID: uuid.New()}, AuthenticatedAt: now.Add(-time.Minute)}},
		},
		{
			name:    "a browser that signed in an hour ago",
			session: Session{Record: model.UserSession{Base: model.Base{ID: uuid.New()}, AuthenticatedAt: now.Add(-time.Hour)}},
			want:    ErrSignInAgain,
		},
		{
			name: "an application's access token",
			want: ErrBrowserRequired,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.session.User = &model.User{}
			if err := s.provePresence(context.Background(), &test.session, "", Client{}); !errors.Is(err, test.want) {
				t.Errorf("provePresence = %v, want %v", err, test.want)
			}
		})
	}
}
