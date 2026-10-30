package model

import (
	"fmt"
	"net/mail"
	"slices"
	"strings"

	"vard/internal/brand"
)

type MailSettings struct {
	Base

	IsEnabled bool `gorm:"not null" json:"is_enabled"`

	Host string `gorm:"size:255;not null;default:''" json:"host"`
	Port int    `gorm:"not null" json:"port"`

	Encryption MailEncryption `gorm:"type:varchar(16);not null;check:chk_mail_settings_encryption,encryption IN ('starttls','tls','none')" json:"encryption"`

	Username string `gorm:"size:255;not null;default:''" json:"username"`
	Password []byte `gorm:"type:bytea" json:"-"`

	FromAddress string `gorm:"size:255;not null;default:''" json:"from_address"`
	FromName    string `gorm:"size:100;not null;default:''" json:"from_name"`
}

func (MailSettings) TableName() string {
	return "mail_settings"
}

type MailEncryption string

const (
	MailStartTLS MailEncryption = "starttls"

	MailTLS MailEncryption = "tls"

	MailNone MailEncryption = "none"
)

var MailEncryptions = []MailEncryption{MailStartTLS, MailTLS, MailNone}

func IsMailEncryption(value MailEncryption) bool {
	return slices.Contains(MailEncryptions, value)
}

// DefaultMailSettings sends nothing: a configured host that does not answer
// would make every sign-up wait on a timeout.
func DefaultMailSettings() MailSettings {
	return MailSettings{
		IsEnabled:   false,
		Port:        587,
		Encryption:  MailStartTLS,
		FromAddress: "no-reply@localhost",
		FromName:    brand.Name,
	}
}

func (m MailSettings) Validate() error {
	if !IsMailEncryption(m.Encryption) {
		return fmt.Errorf("encryption must be one of: %s", joinEncryptions())
	}

	if m.Port < 1 || m.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	if len(m.Host) > 255 {
		return fmt.Errorf("host must be at most 255 characters")
	}
	if strings.ContainsAny(m.Host, " /:") {
		return fmt.Errorf("host must be a host name on its own, such as smtp.example.com")
	}

	if len(m.Username) > 255 {
		return fmt.Errorf("username must be at most 255 characters")
	}
	if len(m.FromName) > 100 {
		return fmt.Errorf("from_name must be at most 100 characters")
	}

	if m.FromAddress != "" {
		if _, err := mail.ParseAddress(m.FromAddress); err != nil {
			return fmt.Errorf("from_address must be an email address")
		}
	}

	if !m.IsEnabled {
		return nil
	}

	switch {
	case m.Host == "":
		return fmt.Errorf("host is required to send email")
	case m.FromAddress == "":
		return fmt.Errorf("from_address is required to send email")
	}

	return nil
}

func joinEncryptions() string {
	names := make([]string, 0, len(MailEncryptions))
	for _, encryption := range MailEncryptions {
		names = append(names, string(encryption))
	}

	return strings.Join(names, ", ")
}

// MailMessageSpec is one email the server sends, for the panel to list. Its
// text lives with the sign-in pages' translations, so it goes out in the
// reader's language.
type MailMessageSpec struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Description string `json:"description"`

	// SubjectKey and BodyKey are the keys in the sign-in pages' catalog.
	SubjectKey string `json:"subject_key"`
	BodyKey    string `json:"body_key"`

	// Params are the {braces} the text may use, so the panel can list them and
	// refuse unknown ones.
	Params []MailParam `json:"params"`
}

// MailParam is one placeholder a message's text may use.
type MailParam struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var MailMessageSpecs = []MailMessageSpec{
	{
		Kind:        "reset",
		Label:       "Password reset",
		Description: "Sent when somebody asks for a new password, with the link that sets one.",
		SubjectKey:  "email.reset.subject",
		BodyKey:     "email.reset.body",
		Params: []MailParam{
			{Name: "app", Description: "The application being signed in to"},
			{Name: "email", Description: "The address the message went to"},
			{Name: "link", Description: "The reset link"},
			{Name: "minutes", Description: "How long the link works, in minutes"},
		},
	},
	{
		Kind:        "verify",
		Label:       "Confirm an address",
		Description: "Sent when a login flow requires a verified address and the account's is not.",
		SubjectKey:  "email.verify.subject",
		BodyKey:     "email.verify.body",
		Params: []MailParam{
			{Name: "app", Description: "The application being signed in to"},
			{Name: "email", Description: "The address the message went to"},
			{Name: "link", Description: "The verification link"},
			{Name: "hours", Description: "How long the link works, in hours"},
		},
	},
	{
		Kind:        "code",
		Label:       "One-time code",
		Description: "Sent by a login flow with the emailed code step, with the code to type back.",
		SubjectKey:  "email.code.subject",
		BodyKey:     "email.code.body",
		Params: []MailParam{
			{Name: "app", Description: "The application being signed in to"},
			{Name: "email", Description: "The address the message went to"},
			{Name: "code", Description: "The code itself"},
			{Name: "minutes", Description: "How long the code works, in minutes"},
		},
	},
}

func MailMessageKeys() []string {
	keys := make([]string, 0, len(MailMessageSpecs)*2)
	for _, spec := range MailMessageSpecs {
		keys = append(keys, spec.SubjectKey, spec.BodyKey)
	}

	return keys
}

const MailTextPrefix = "email."

// IsMailTextKey keeps translators' saves off email text, where {link} places a
// live reset token.
func IsMailTextKey(key string) bool {
	return strings.HasPrefix(key, MailTextPrefix)
}

// IsMailMessageKey reports whether a key is one of them, so a write to the
// Mail page cannot reach the rest of a language's text.
func IsMailMessageKey(key string) bool {
	return slices.Contains(MailMessageKeys(), key)
}
