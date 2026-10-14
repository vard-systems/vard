package mail

import (
	"mime"
	"net/mail"
	"strings"
	"testing"

	"loginer/internal/brand"
	"loginer/internal/config"
	"loginer/internal/model"
)

// A message has to leave here as a mail client expects to find it: headers
// separated from the body by a blank line, every line ending CRLF, and a
// subject that survives being written in any script.
func TestCompose(t *testing.T) {
	from := &mail.Address{Name: "Acme", Address: "no-reply@acme.example"}
	to := &mail.Address{Address: "ada@example.com"}

	composed := string(compose(from, to, Message{
		To:      to.Address,
		Subject: "Сброс пароля",
		Body:    "Line one\nLine two\n",
	}))

	headers, body, split := strings.Cut(composed, "\r\n\r\n")
	if !split {
		t.Fatalf("no blank line between the headers and the body:\n%q", composed)
	}

	for _, want := range []string{
		`From: "Acme" <no-reply@acme.example>`,
		"To: <ada@example.com>",
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="utf-8"`,
	} {
		if !strings.Contains(headers, want) {
			t.Errorf("headers are missing %q:\n%s", want, headers)
		}
	}

	// The subject is encoded rather than sent as raw UTF-8: a header is
	// ASCII, and an application's name can be in any script.
	if strings.Contains(headers, "Сброс") {
		t.Errorf("the subject was not encoded:\n%s", headers)
	}
	subject, err := (&mime.WordDecoder{}).DecodeHeader(headerValue(headers, "Subject"))
	if err != nil {
		t.Fatal(err)
	}
	if subject != "Сброс пароля" {
		t.Errorf("the subject decodes to %q, want the one that was sent", subject)
	}

	// Every line of the body ends CRLF, including the ones the caller wrote
	// with a bare newline.
	for _, line := range strings.SplitAfter(body, "\n") {
		if line != "" && !strings.HasSuffix(line, "\r\n") {
			t.Errorf("a body line does not end CRLF: %q", line)
		}
	}
}

// headerValue is one header's value, from the block compose wrote.
func headerValue(headers, name string) string {
	for line := range strings.SplitSeq(headers, "\r\n") {
		if after, found := strings.CutPrefix(line, name+": "); found {
			return after
		}
	}

	return ""
}

// What a fresh installation's mail settings are, read from the configuration
// the server was started with.
func TestSeed(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Mail
		want model.MailSettings
	}{
		{
			name: "nothing configured sends nothing",
			cfg:  config.Mail{Port: 587, From: brand.Name + " <no-reply@localhost>"},
			want: model.MailSettings{
				IsEnabled: false, Port: 587, Encryption: model.MailStartTLS,
				FromAddress: "no-reply@localhost", FromName: brand.Name,
			},
		},
		{
			name: "a host named turns sending on",
			cfg: config.Mail{
				Host: "smtp.example.com", Port: 587, Username: "apikey",
				From: "Acme <no-reply@acme.example>",
			},
			want: model.MailSettings{
				IsEnabled: true, Host: "smtp.example.com", Port: 587,
				Encryption: model.MailStartTLS, Username: "apikey",
				FromAddress: "no-reply@acme.example", FromName: "Acme",
			},
		},
		{
			name: "the port that expects TLS from the first byte",
			cfg:  config.Mail{Host: "smtp.example.com", Port: 465, From: "a@b.example"},
			want: model.MailSettings{
				IsEnabled: true, Host: "smtp.example.com", Port: 465,
				Encryption: model.MailTLS, FromAddress: "a@b.example",
			},
		},
		{
			name: "an address that is not one is left for the Mail page",
			cfg:  config.Mail{Host: "smtp.example.com", Port: 25, From: "no-reply at acme"},
			want: model.MailSettings{
				IsEnabled: true, Host: "smtp.example.com", Port: 25,
				Encryption: model.MailStartTLS,
				// The default's, since nothing could be read from the file.
				FromAddress: "no-reply@localhost", FromName: brand.Name,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, password := Seed(tt.cfg)

			if password != tt.cfg.Password {
				t.Errorf("password = %q, want the configured one", password)
			}

			switch {
			case got.IsEnabled != tt.want.IsEnabled:
				t.Errorf("enabled = %v, want %v", got.IsEnabled, tt.want.IsEnabled)
			case got.Host != tt.want.Host:
				t.Errorf("host = %q, want %q", got.Host, tt.want.Host)
			case got.Port != tt.want.Port:
				t.Errorf("port = %d, want %d", got.Port, tt.want.Port)
			case got.Encryption != tt.want.Encryption:
				t.Errorf("encryption = %q, want %q", got.Encryption, tt.want.Encryption)
			case got.Username != tt.want.Username:
				t.Errorf("username = %q, want %q", got.Username, tt.want.Username)
			case got.FromAddress != tt.want.FromAddress:
				t.Errorf("from_address = %q, want %q", got.FromAddress, tt.want.FromAddress)
			case got.FromName != tt.want.FromName:
				t.Errorf("from_name = %q, want %q", got.FromName, tt.want.FromName)
			}

			// Whatever the configuration said, the row a fresh installation
			// starts with has to be one the panel would accept.
			if got.IsEnabled {
				if err := got.Validate(); err != nil {
					t.Errorf("the seeded settings are invalid: %v", err)
				}
			}
		})
	}
}

// The settings a stored record becomes, with its password already unsealed.
func TestOf(t *testing.T) {
	stored := model.MailSettings{
		IsEnabled: true, Host: "smtp.example.com", Port: 465,
		Encryption: model.MailTLS, Username: "apikey",
		FromAddress: "no-reply@acme.example", FromName: "Acme",
	}

	settings := Of(stored, "a secret")

	if settings.Address() != "smtp.example.com:465" {
		t.Errorf("address = %q, want host and port", settings.Address())
	}
	if settings.Encryption != TLS {
		t.Errorf("encryption = %q, want %q", settings.Encryption, TLS)
	}
	if settings.Password != "a secret" {
		t.Errorf("password = %q, want the one handed in", settings.Password)
	}
	if got := settings.From().String(); got != `"Acme" <no-reply@acme.example>` {
		t.Errorf("from = %s, want the name in front of the address", got)
	}
}
