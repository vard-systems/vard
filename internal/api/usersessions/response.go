package usersessions

import (
	"strings"
	"time"

	"vard/internal/store"
)

type sessionUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// sessionResponse is one session as the panel lists it.
type sessionResponse struct {
	ID         string      `json:"id"`
	User       sessionUser `json:"user"`
	IP         string      `json:"ip"`
	UserAgent  string      `json:"user_agent"`
	SignedInAt time.Time   `json:"signed_in_at"`
	ExpiresAt  time.Time   `json:"expires_at"`
}

// listResponse is a page of sessions and a cursor, never a total.
type listResponse struct {
	Sessions []sessionResponse `json:"sessions"`
	Next     string            `json:"next,omitempty"`
}

func newSessionResponse(session store.ActiveSession) sessionResponse {
	return sessionResponse{
		ID: session.ID.String(),
		User: sessionUser{
			ID:    session.UserID.String(),
			Email: session.Email,
			Name:  strings.TrimSpace(session.FirstName + " " + session.LastName),
		},
		IP:         session.IP,
		UserAgent:  session.UserAgent,
		SignedInAt: session.AuthenticatedAt,
		ExpiresAt:  session.ExpiresAt,
	}
}
