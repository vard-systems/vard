package api

import (
	"maps"
	"net/http"
	"testing"

	"vard/internal/domain/model"
)

// The emails are a super admin's, on the Mail page: the reset email puts a
// live reset link wherever its body says {link}. A translator — somebody with
// languages.write and nothing more — saves the rest of a language's text and
// leaves the emails as they were.
func TestLiveTranslatorsCannotRewordEmails(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	super.must(http.StatusCreated, http.MethodPost, "/admin-roles", map[string]any{
		"name": "translator", "permissions": []string{model.PermLanguagesRead, model.PermLanguagesWrite},
	}, nil)
	const email, password = "translator@example.com", "translator-password-1"
	super.must(http.StatusCreated, http.MethodPost, "/admins", map[string]any{
		"email": email, "first_name": "Trans", "status": "active",
		"password": password, "confirm_password": password,
		"assignments": []map[string]any{{"role_id": super.adminRoleID("translator")}},
	}, nil)
	translator := s.client()
	if status := translator.login(email, password); status != http.StatusOK {
		t.Fatalf("translator login = %d", status)
	}

	var text struct {
		Base     map[string]string `json:"base"`
		Messages map[string]string `json:"messages"`
	}
	super.must(http.StatusOK, http.MethodGet, "/languages/en/translations/id", nil, &text)
	if text.Base["email.reset.body"] == "" {
		t.Fatal("the shipped English has no reset email to protect")
	}

	// The whole draft, as the editor sends it, with one email and one page
	// reworded.
	const reworded = "Your account is at risk. Secure it at https://evil.example/r?u={link}"
	draft := map[string]string{}
	maps.Copy(draft, text.Base)
	draft["email.reset.body"] = reworded
	draft["action.sign_in"] = "Sign in now"
	translator.must(http.StatusOK, http.MethodPut, "/languages/en/translations/id", map[string]any{"messages": draft}, nil)

	super.must(http.StatusOK, http.MethodGet, "/languages/en/translations/id", nil, &text)
	if text.Messages["email.reset.body"] == reworded {
		t.Errorf("a translator reworded the reset email to %q", reworded)
	}
	if text.Messages["action.sign_in"] != "Sign in now" {
		t.Errorf("action.sign_in = %q, want the translator's wording saved", text.Messages["action.sign_in"])
	}

	// A super admin rewords it, as the Mail page does.
	draft["email.reset.body"] = "Choose a new password: {link}"
	super.must(http.StatusOK, http.MethodPut, "/languages/en/translations/id", map[string]any{"messages": draft}, nil)
	super.must(http.StatusOK, http.MethodGet, "/languages/en/translations/id", nil, &text)
	if text.Messages["email.reset.body"] != "Choose a new password: {link}" {
		t.Errorf("a super admin's rewording = %q, want it saved", text.Messages["email.reset.body"])
	}
}
