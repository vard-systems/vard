package mailsettings

import (
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/jose"
)

// sealer is the one these tests seal a password with. The key is only ever
// used here.
func sealer(t *testing.T) *jose.Sealer {
	t.Helper()

	s, err := jose.NewSealer("a test key that is long enough to be one")
	if err != nil {
		t.Fatalf("NewSealer() = %v", err)
	}

	return s
}

// stored is the mail server a request in these tests is applied to.
func stored(t *testing.T) model.MailSettings {
	t.Helper()

	settings := model.MailSettings{Base: model.Base{ID: uuid.New()}}
	password, err := sealer(t).SealFor(settings.PasswordWhere(), []byte("the stored password"))
	if err != nil {
		t.Fatalf("SealFor() = %v", err)
	}

	return model.MailSettings{
		Base:        settings.Base,
		IsEnabled:   true,
		Host:        "smtp.example.com",
		Port:        587,
		Encryption:  model.MailStartTLS,
		Username:    "apikey",
		Password:    password,
		FromAddress: "no-reply@example.com",
		FromName:    "Acme",
	}
}

func TestSettingsRequestApplyTo(t *testing.T) {
	tests := []struct {
		name    string
		request settingsRequest
		want    string   // the message, or "" when the request is fine
		changed []string // the settings it should report as changed
		check   func(model.MailSettings) error
	}{
		{
			name:    "nothing sent changes nothing",
			request: settingsRequest{},
			changed: []string{},
		},
		{
			name:    "a new host, trimmed and lower case",
			request: settingsRequest{Host: new("  SMTP.Example.NET  ")},
			changed: []string{"host"},
			check: func(m model.MailSettings) error {
				if m.Host != "smtp.example.net" {
					return errors.New("host = " + m.Host)
				}
				return nil
			},
		},
		{
			name:    "the same values again are no change",
			request: settingsRequest{Host: new("smtp.example.com"), Port: new(587)},
			changed: []string{},
		},
		{
			name:    "moving to the port that expects TLS from the first byte",
			request: settingsRequest{Port: new(465), Encryption: new("tls")},
			changed: []string{"port", "encryption"},
		},
		{
			name:    "turning sending off leaves the server as it is",
			request: settingsRequest{IsEnabled: new(false)},
			changed: []string{"is_enabled"},
			check: func(m model.MailSettings) error {
				if m.Host != "smtp.example.com" {
					return errors.New("the host was cleared: " + m.Host)
				}
				return nil
			},
		},
		{
			name:    "a password left out is the stored one",
			request: settingsRequest{Username: new("someone")},
			changed: []string{"username"},
			check: func(m model.MailSettings) error {
				if len(m.Password) == 0 {
					return errors.New("the stored password was cleared")
				}
				return nil
			},
		},
		{
			name:    "a new password is sealed, not stored as it was typed",
			request: settingsRequest{Password: new("a new password")},
			changed: []string{"password"},
			check: func(m model.MailSettings) error {
				if string(m.Password) == "a new password" {
					return errors.New("the password was stored in the clear")
				}
				if len(m.Password) == 0 {
					return errors.New("the password was not stored")
				}
				return nil
			},
		},
		{
			name:    "an empty password is a server that takes none",
			request: settingsRequest{Password: new("")},
			changed: []string{"password"},
			check: func(m model.MailSettings) error {
				if len(m.Password) != 0 {
					return errors.New("the password was kept")
				}
				return nil
			},
		},
		{
			name:    "sending with no host",
			request: settingsRequest{Host: new("")},
			want:    "host is required to send email",
		},
		{
			name:    "a host with the port in it",
			request: settingsRequest{Host: new("smtp.example.com:587")},
			want:    "host must be a host name on its own, such as smtp.example.com",
		},
		{
			name:    "an encryption nothing speaks",
			request: settingsRequest{Encryption: new("ssl")},
			want:    "encryption must be one of: starttls, tls, none",
		},
		{
			name:    "an address that is not one",
			request: settingsRequest{FromAddress: new("no-reply at example")},
			want:    "from_address must be an email address",
		},
		{
			name:    "a port nothing listens on",
			request: settingsRequest{Port: new(0)},
			want:    "port must be between 1 and 65535",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			settings := stored(t)
			before := stored(t)

			changed, err := request.applyTo(&settings, sealer(t))

			if tt.want != "" {
				var fault respond.Fault
				if !errors.As(err, &fault) {
					t.Fatalf("applyTo() = %v, want a Fault", err)
				}
				if fault.Message != tt.want {
					t.Errorf("message = %q, want %q", fault.Message, tt.want)
				}
				// Nothing may be written when a request is refused.
				if settings.Host != before.Host || settings.Port != before.Port {
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
			if tt.check != nil {
				if err := tt.check(settings); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// The password never leaves the server: the panel is told whether there is
// one, and nothing else.
func TestResponseKeepsThePasswordIn(t *testing.T) {
	answer := newResponse(stored(t))

	if !answer.Mail.HasPassword {
		t.Errorf("has_password = false, want true: one is stored")
	}

	empty := newResponse(model.DefaultMailSettings())
	if empty.Mail.HasPassword {
		t.Errorf("has_password = true, want false: none is stored")
	}

	if len(answer.Encryptions) != len(model.MailEncryptions) {
		t.Errorf("encryptions = %d, want the model's %d", len(answer.Encryptions), len(model.MailEncryptions))
	}
}

// A test message is sent with what the form holds, so a server can be tried
// before it is saved — and with the stored password where none was typed.
func TestTestRequestSettings(t *testing.T) {
	saved := stored(t)

	t.Run("the form's server, the stored password", func(t *testing.T) {
		req := testRequest{To: "someone@example.com",
			Host: new("smtp.other.example")}

		settings, err := req.settings(&saved, sealer(t))
		if err != nil {
			t.Fatalf("settings() = %v", err)
		}

		if settings.Host != "smtp.other.example" {
			t.Errorf("host = %q, want the form's", settings.Host)
		}
		if settings.Password != "the stored password" {
			t.Errorf("password = %q, want the stored one unsealed", settings.Password)
		}
	})

	t.Run("a password just typed is the one tried", func(t *testing.T) {
		req := testRequest{To: "someone@example.com",
			Password: new("a new password")}

		settings, err := req.settings(&saved, sealer(t))
		if err != nil {
			t.Fatalf("settings() = %v", err)
		}

		if settings.Password != "a new password" {
			t.Errorf("password = %q, want the one on the form", settings.Password)
		}
	})

	t.Run("a test is sent even with sending switched off", func(t *testing.T) {
		off := stored(t)
		off.IsEnabled = false

		req := testRequest{To: "someone@example.com"}

		settings, err := req.settings(&off, sealer(t))
		if err != nil {
			t.Fatalf("settings() = %v", err)
		}

		if !settings.Enabled {
			t.Errorf("enabled = false: a test would be logged rather than sent")
		}
	})

	t.Run("a server that could not work is refused before it is dialled", func(t *testing.T) {
		req := testRequest{To: "someone@example.com",
			Host: new("")}

		_, err := req.settings(&saved, sealer(t))

		if _, ok := errors.AsType[respond.Fault](err); !ok {
			t.Fatalf("settings() = %v, want a Fault", err)
		}
	})
}

func TestContentRequestValidate(t *testing.T) {
	tests := []struct {
		name     string
		messages map[string]string
		wantCode string // the problem's code, or "" when the request is fine
	}{
		{
			name:     "the words of an email",
			messages: map[string]string{"email.reset.subject": "Reset your password"},
		},
		{
			name:     "an empty value, which clears the override",
			messages: map[string]string{"email.code.body": ""},
		},
		{
			name:     "a key that is not an email's",
			messages: map[string]string{"login.title": "Sign in"},
			wantCode: "mail_content_key_unknown",
		},
		{
			name:     "a key nothing has",
			messages: map[string]string{"email.nothing.subject": "Hello"},
			wantCode: "mail_content_key_unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := contentRequest{Messages: tt.messages}

			err := req.validate()

			if tt.wantCode == "" {
				if err != nil {
					t.Fatalf("validate() = %v, want nothing", err)
				}
				return
			}

			var fault respond.Fault
			if !errors.As(err, &fault) {
				t.Fatalf("validate() = %v, want a Fault", err)
			}
			if fault.Code != tt.wantCode {
				t.Errorf("code = %q, want %q", fault.Code, tt.wantCode)
			}
		})
	}
}
