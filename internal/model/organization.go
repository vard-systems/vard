package model

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"

	"loginer/internal/brand"
)

type Organization struct {
	Base

	Name string `gorm:"size:100;not null" json:"name"`

	Slug string `gorm:"size:64;not null;uniqueIndex" json:"slug"`

	LogoURL string `gorm:"size:512;not null;default:''" json:"logo_url"`

	SupportEmail string `gorm:"size:255;not null;default:''" json:"support_email"`
	SupportPhone string `gorm:"size:32;not null;default:''" json:"support_phone"`

	TermsURL   string `gorm:"size:512;not null;default:''" json:"terms_url"`
	PrivacyURL string `gorm:"size:512;not null;default:''" json:"privacy_url"`
	Timezone   string `gorm:"size:64;not null;default:UTC" json:"timezone"`

	// AgreementsUpdatedAt is when the terms or privacy link last changed. A
	// user who accepted them before then is asked again by the Agreements
	// step.
	AgreementsUpdatedAt *time.Time `json:"agreements_updated_at"`
}

// HasAgreements says there is something for the Agreements step to ask.
func (o Organization) HasAgreements() bool {
	return o.TermsURL != "" || o.PrivacyURL != ""
}

// AgreementsAccepted says a user who accepted at `at` (nil: never) has
// accepted the agreements as they stand.
func (o Organization) AgreementsAccepted(at *time.Time) bool {
	if !o.HasAgreements() {
		return true
	}
	if at == nil {
		return false
	}

	return o.AgreementsUpdatedAt == nil || !at.Before(*o.AgreementsUpdatedAt)
}

func (Organization) TableName() string {
	return "organizations"
}

func DefaultOrganization() Organization {
	return Organization{Name: brand.Name, Slug: brand.Slug, Timezone: "UTC"}
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

var phonePattern = regexp.MustCompile(`^\+?[0-9][0-9 ()./-]{4,30}$`)

func (o Organization) Validate() error {
	switch {
	case strings.TrimSpace(o.Name) == "":
		return fmt.Errorf("name is required")
	case len(o.Name) > 100:
		return fmt.Errorf("name must be at most 100 characters")
	}

	switch {
	case o.Slug == "":
		return fmt.Errorf("slug is required")
	case len(o.Slug) > 64:
		return fmt.Errorf("slug must be at most 64 characters")
	case !slugPattern.MatchString(o.Slug):
		return fmt.Errorf("slug must be lower case letters, numbers and dashes, such as acme-inc")
	}

	if err := validTimezone(o.Timezone); err != nil {
		return err
	}

	if o.SupportEmail != "" {
		if _, err := mail.ParseAddress(o.SupportEmail); err != nil {
			return fmt.Errorf("support_email must be an email address")
		}
	}

	if o.SupportPhone != "" {
		if len(o.SupportPhone) > 32 || !phonePattern.MatchString(o.SupportPhone) {
			return fmt.Errorf("support_phone must be a number someone can dial, such as +996 555 123456")
		}
	}

	for _, link := range []struct {
		field string
		value string
	}{
		{"logo_url", o.LogoURL},
		{"terms_url", o.TermsURL},
		{"privacy_url", o.PrivacyURL},
	} {
		if err := ValidLink(link.field, link.value); err != nil {
			return err
		}
	}

	return nil
}

// validTimezone requires an IANA zone; "Local" is refused because it depends on
// the host.
func validTimezone(zone string) error {
	if zone == "" {
		return fmt.Errorf("timezone is required")
	}

	if _, err := time.LoadLocation(zone); err != nil || zone == "Local" || len(zone) > 64 {
		return fmt.Errorf("timezone must be a zone such as Asia/Bishkek or UTC")
	}

	return nil
}

// ValidLink accepts an empty value or an absolute http(s) URL; these are
// rendered in pages, where a relative path or javascript: URL would be wrong or
// dangerous.
func ValidLink(field, value string) error {
	if value == "" {
		return nil
	}

	if len(value) > 512 {
		return fmt.Errorf("%s must be at most 512 characters", field)
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be a full address starting with http:// or https://", field)
	}

	return nil
}
