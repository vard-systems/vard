package api

import (
	"net/http"
	"testing"
)

// The Sessions page: who is signed in, found by the start of an address and
// paged without counting, and signed out — one browser, or everywhere.
func TestLiveSessions(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	for _, email := range []string{"grace@example.com", "alan@example.com"} {
		super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
			"email": email, "is_email_verified": true, "password": "a-password-1", "confirm_password": "a-password-1",
		}, nil)
	}

	signIn := func(email string) *browser {
		b := s.browser()
		if status := b.account(http.MethodPost, "/login", map[string]string{"email": email, "password": "a-password-1"}, nil); status != http.StatusOK {
			t.Fatalf("signing %s in = %d", email, status)
		}
		return b
	}
	laptop, phone := signIn("grace@example.com"), signIn("grace@example.com")
	signIn("alan@example.com")

	type page struct {
		Sessions []struct {
			ID   string `json:"id"`
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
		} `json:"sessions"`
		Next string `json:"next"`
	}
	list := func(query string) page {
		var out page
		super.must(http.StatusOK, http.MethodGet, "/user-sessions"+query, nil, &out)
		return out
	}

	for _, tt := range []struct {
		name, query string
		want        int
	}{
		{name: "everyone", query: "", want: 3},
		{name: "the start of an address", query: "?search=GRA", want: 2},
		{name: "an address nobody has", query: "?search=nobody", want: 0},
	} {
		if got := len(list(tt.query).Sessions); got != tt.want {
			t.Errorf("%s: %d sessions, want %d", tt.name, got, tt.want)
		}
	}

	// Paged by the last one shown, newest first, until there is no next.
	first := list("?search=grace&limit=1")
	if len(first.Sessions) != 1 || first.Next == "" {
		t.Fatalf("the first page = %+v, want one session and a next", first)
	}
	second := list("?search=grace&limit=1&after=" + first.Next)
	if len(second.Sessions) != 1 || second.Next != "" || second.Sessions[0].ID == first.Sessions[0].ID {
		t.Fatalf("the second page = %+v, want the other session and no next", second)
	}
	graceID := first.Sessions[0].User.ID

	if got := len(list("?user=" + graceID).Sessions); got != 2 {
		t.Errorf("one user's sessions = %d, want 2", got)
	}

	// Ending the newest session signs that browser out and no other.
	var refused problemBody
	if status := super.do(http.MethodDelete, "/user-sessions/00000000-0000-0000-0000-000000000000", nil, &refused); status != http.StatusNotFound || refused.Code != "session_not_found" {
		t.Errorf("ending a session nobody has = %d %+v, want session_not_found", status, refused)
	}

	super.must(http.StatusNoContent, http.MethodDelete, "/user-sessions/"+first.Sessions[0].ID, nil, nil)
	signedIn := 0
	for _, b := range []*browser{laptop, phone} {
		if b.account(http.MethodGet, "/me", nil, nil) == http.StatusOK {
			signedIn++
		}
	}
	if signedIn != 1 {
		t.Errorf("after ending one of two sessions, %d browsers are signed in, want 1", signedIn)
	}

	// Everywhere: every session of hers ends, and Alan's does not.
	var out struct {
		Sessions int64 `json:"sessions"`
	}
	super.must(http.StatusOK, http.MethodDelete, "/users/"+graceID+"/sessions", nil, &out)
	if out.Sessions != 1 {
		t.Errorf("signing Grace out everywhere ended %d sessions, want the one left", out.Sessions)
	}
	for _, b := range []*browser{laptop, phone} {
		if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
			t.Errorf("a browser of hers after signing out everywhere = %d, want 401", status)
		}
	}
	if got := list(""); len(got.Sessions) != 1 || got.Sessions[0].User.Email != "alan@example.com" {
		t.Errorf("sessions left = %+v, want Alan's alone", got.Sessions)
	}

	var logs struct {
		Logs []struct {
			Action string `json:"action"`
		} `json:"logs"`
	}
	super.must(http.StatusOK, http.MethodGet, "/logs?limit=100", nil, &logs)
	recorded := map[string]bool{}
	for _, entry := range logs.Logs {
		recorded[entry.Action] = true
	}
	for _, action := range []string{"user.session_ended", "user.signed_out_everywhere"} {
		if !recorded[action] {
			t.Errorf("%s is not in the activity log", action)
		}
	}
}
