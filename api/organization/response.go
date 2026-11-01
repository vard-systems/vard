package organization

import (
	"time"

	"vard/core/model"
)

// organizationResponse is built by hand to keep row bookkeeping out of a
// settings record.
type organizationResponse struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	// LogoVersion names the uploaded app logo, empty while there is none;
	// the panel loads it by its version.
	LogoVersion  string    `json:"logo_version"`
	SupportEmail string    `json:"support_email"`
	SupportPhone string    `json:"support_phone"`
	TermsURL     string    `json:"terms_url"`
	PrivacyURL   string    `json:"privacy_url"`
	Timezone     string    `json:"timezone"`
	CreatedAt    time.Time `json:"created_at"`
}

type response struct {
	Organization organizationResponse `json:"organization"`
}

func newResponse(organization model.Organization) response {
	return response{
		Organization: organizationResponse{
			Name:         organization.Name,
			Slug:         organization.Slug,
			LogoVersion:  organization.LogoVersion,
			SupportEmail: organization.SupportEmail,
			SupportPhone: organization.SupportPhone,
			TermsURL:     organization.TermsURL,
			PrivacyURL:   organization.PrivacyURL,
			Timezone:     organization.Timezone,
			CreatedAt:    organization.CreatedAt,
		},
	}
}
