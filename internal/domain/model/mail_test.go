package model

import "testing"

// settings is a mail server that is configured and works, which each test
// then breaks in one way.
func mailSettings() MailSettings {
	return MailSettings{
		IsEnabled:   true,
		Host:        "smtp.example.com",
		Port:        587,
		Encryption:  MailStartTLS,
		Username:    "apikey",
		FromAddress: "no-reply@example.com",
		FromName:    "Acme",
	}
}

func TestMailSettingsValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(*MailSettings)
		want   string // the message, or "" when the settings are fine
	}{
		{
			name:   "a working mail server",
			change: func(*MailSettings) {},
		},
		{
			name:   "nothing configured, and nothing sent",
			change: func(m *MailSettings) { *m = DefaultMailSettings() },
		},
		{
			name: "a half-filled form with sending off is somebody's work in progress",
			change: func(m *MailSettings) {
				m.IsEnabled = false
				m.Host = ""
				m.FromAddress = ""
			},
		},
		{
			name:   "sending with no host",
			change: func(m *MailSettings) { m.Host = "" },
			want:   "host is required to send email",
		},
		{
			name:   "sending with nothing to send from",
			change: func(m *MailSettings) { m.FromAddress = "" },
			want:   "from_address is required to send email",
		},
		{
			name:   "a host with the port in it",
			change: func(m *MailSettings) { m.Host = "smtp.example.com:587" },
			want:   "host must be a host name on its own, such as smtp.example.com",
		},
		{
			name:   "a host with a scheme",
			change: func(m *MailSettings) { m.Host = "smtps://smtp.example.com" },
			want:   "host must be a host name on its own, such as smtp.example.com",
		},
		{
			name:   "a port nothing listens on",
			change: func(m *MailSettings) { m.Port = 0 },
			want:   "port must be between 1 and 65535",
		},
		{
			name:   "a port past the end of them",
			change: func(m *MailSettings) { m.Port = 70000 },
			want:   "port must be between 1 and 65535",
		},
		{
			name:   "an encryption nothing speaks",
			change: func(m *MailSettings) { m.Encryption = "ssl" },
			want:   "encryption must be one of: starttls, tls, none",
		},
		{
			name:   "an address that is not one",
			change: func(m *MailSettings) { m.FromAddress = "no-reply at example" },
			want:   "from_address must be an email address",
		},
		{
			name: "an address that is not one, even with sending off",
			change: func(m *MailSettings) {
				m.IsEnabled = false
				m.FromAddress = "no-reply at example"
			},
			want: "from_address must be an email address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := mailSettings()
			tt.change(&settings)

			err := settings.Validate()

			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nothing", err)
			case tt.want == "":
				return
			case err == nil:
				t.Fatalf("Validate() = nothing, want %q", tt.want)
			case err.Error() != tt.want:
				t.Errorf("Validate() = %q, want %q", err, tt.want)
			}
		})
	}
}

// Every email the server sends has to be in the catalog the Mail page is
// built from, and each of its keys has to be one the sign-in pages look up —
// a key that is in neither is text nobody can edit, or edits nobody reads.
func TestMailMessageSpecsNameRealKeys(t *testing.T) {
	seen := map[string]bool{}

	for _, spec := range MailMessageSpecs {
		if spec.Kind == "" || spec.Label == "" {
			t.Errorf("a mail message spec has no kind or label: %+v", spec)
		}
		if seen[spec.Kind] {
			t.Errorf("%s is listed twice", spec.Kind)
		}
		seen[spec.Kind] = true

		for _, key := range []string{spec.SubjectKey, spec.BodyKey} {
			if !IsMailMessageKey(key) {
				t.Errorf("%s names %s, which MailMessageKeys does not list", spec.Kind, key)
			}
		}
	}

	if got, want := len(MailMessageKeys()), len(MailMessageSpecs)*2; got != want {
		t.Errorf("MailMessageKeys() has %d keys, want %d — a subject and a body each", got, want)
	}
}

// Every key an email is built from is email text — the messages the Mail page
// edits and the words put into them alike — and nothing else is.
func TestIsMailTextKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{name: "the reset email's body, which carries the link", key: "email.reset.body", want: true},
		{name: "a message's subject", key: "email.code.subject", want: true},
		{name: "a word put into a message", key: "email.reset.your_account", want: true},
		{name: "the sign-in page's text", key: "action.sign_in", want: false},
		{name: "a key that merely starts with the word", key: "email_change.title", want: false},
		{name: "the language's own name", key: "$name", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMailTextKey(tt.key); got != tt.want {
				t.Errorf("IsMailTextKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}

	for _, key := range MailMessageKeys() {
		if !IsMailTextKey(key) {
			t.Errorf("%s is a Mail page key but not email text", key)
		}
	}
}
