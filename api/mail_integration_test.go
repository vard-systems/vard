package api

import (
	"net/http"
	"testing"
)

// The mail settings are a super admin's, the password never comes back, and
// the words of an email are the sign-in pages' text — so writing them on the
// Mail page is what the next message says.
func TestLiveMailSettingsAreSuperAdmins(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	var settings struct {
		Mail struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			HasPassword bool   `json:"has_password"`
			Password    string `json:"password"`
		} `json:"mail"`
		Encryptions []string `json:"encryptions"`
	}
	super.must(http.StatusOK, http.MethodGet, "/mail", nil, &settings)

	if len(settings.Encryptions) == 0 {
		t.Error("the page was given no encryptions to offer")
	}

	super.must(http.StatusOK, http.MethodPatch, "/mail", map[string]any{
		"is_enabled": true, "host": "SMTP.Example.COM", "port": 465, "encryption": "tls",
		"username": "apikey", "password": "a secret", "from_address": "no-reply@example.com",
	}, &settings)

	if settings.Mail.Host != "smtp.example.com" {
		t.Errorf("host = %q, want it stored lower case", settings.Mail.Host)
	}
	if !settings.Mail.HasPassword {
		t.Error("has_password = false after one was set")
	}
	if settings.Mail.Password != "" {
		t.Errorf("the password came back from the API: %q", settings.Mail.Password)
	}

	// Reading it again says one is stored and still does not say what it is.
	super.must(http.StatusOK, http.MethodGet, "/mail", nil, &settings)
	if settings.Mail.Password != "" || !settings.Mail.HasPassword {
		t.Errorf("mail = %+v, want a stored password and no sight of it", settings.Mail)
	}

	// A host that could not work is refused rather than stored.
	var refused problemBody
	if status := super.do(http.MethodPatch, "/mail", map[string]any{"host": "smtp.example.com:465"}, &refused); status != http.StatusBadRequest {
		t.Errorf("a host with a port in it = %d %+v, want 400", status, refused)
	}

	// The words of the emails, which are the sign-in pages' text.
	var content struct {
		Messages []struct {
			Kind       string `json:"kind"`
			SubjectKey string `json:"subject_key"`
		} `json:"messages"`
		Languages []struct {
			Code string            `json:"code"`
			Base map[string]string `json:"base"`
		} `json:"languages"`
	}
	super.must(http.StatusOK, http.MethodGet, "/mail/content", nil, &content)

	if len(content.Messages) == 0 || len(content.Languages) == 0 {
		t.Fatalf("content = %+v, want the messages and the languages", content)
	}
	if content.Languages[0].Base["email.reset.subject"] == "" {
		t.Error("the shipped English for a reset email is missing")
	}

	super.must(http.StatusOK, http.MethodPut, "/mail/content/en", map[string]any{
		"messages": map[string]string{"email.reset.subject": "Choose a new password"},
	}, nil)

	// A key that is not an email's cannot be written through this page.
	var unknown problemBody
	if status := super.do(http.MethodPut, "/mail/content/en", map[string]any{
		"messages": map[string]string{"login.title": "Sign in"},
	}, &unknown); status != http.StatusBadRequest || unknown.Code != "mail_content_key_unknown" {
		t.Errorf("a key that is not an email's = %d %+v, want mail_content_key_unknown", status, unknown)
	}

	// The words that were written are what the next message actually says.
	super.must(http.StatusCreated, http.MethodPost, "/users", map[string]any{
		"email": "alan@example.com", "password": "alan-password-1", "confirm_password": "alan-password-1",
	}, nil)
	s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": "alan@example.com"}, nil)

	if subject := s.mail.wait(t, "alan@example.com").Subject; subject != "Choose a new password" {
		t.Errorf("the reset email's subject = %q, want the words written on the Mail page", subject)
	}
}
