package oidc

import (
	"context"

	"vard/internal/model"
)

// PublicOrganization is what the sign-in pages show about the organisation. It
// is built by hand because anyone can read it.
type PublicOrganization struct {
	Name string `json:"name"`
	// LogoURL is the app logo, on the sign-in pages' own origin; empty while
	// there is none.
	LogoURL      string `json:"logo_url"`
	SupportEmail string `json:"support_email"`
	SupportPhone string `json:"support_phone"`
	TermsURL     string `json:"terms_url"`
	PrivacyURL   string `json:"privacy_url"`
	// Timezone is the zone the pages say dates in.
	Timezone string `json:"timezone"`
}

func publicOrganization(organization *model.Organization) PublicOrganization {
	return PublicOrganization{
		Name:         organization.Name,
		LogoURL:      logoURL(organization),
		SupportEmail: organization.SupportEmail,
		SupportPhone: organization.SupportPhone,
		TermsURL:     organization.TermsURL,
		PrivacyURL:   organization.PrivacyURL,
		Timezone:     organization.Timezone,
	}
}

// PathOrganizationLogo serves the organisation's app logo to anyone: the
// sign-in pages show it before anybody signs in.
const PathOrganizationLogo = "/api/v1/account/organization/logo"

func logoURL(organization *model.Organization) string {
	if organization.LogoVersion == "" {
		return ""
	}

	return PathOrganizationLogo + "?v=" + organization.LogoVersion
}

// OrganizationLogo is the app logo, or store.ErrNotFound.
func (s *Service) OrganizationLogo(ctx context.Context) (*model.OrganizationLogo, error) {
	return s.store.OrganizationLogo(ctx)
}

func (s *Service) Organization(ctx context.Context) (PublicOrganization, error) {
	organization, err := s.store.Organization(ctx)
	if err != nil {
		return PublicOrganization{}, err
	}

	return publicOrganization(organization), nil
}
