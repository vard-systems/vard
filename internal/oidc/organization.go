package oidc

import (
	"context"

	"loginer/internal/model"
)

// PublicOrganization is what the sign-in pages show about the organisation. It
// is built by hand because anyone can read it.
type PublicOrganization struct {
	Name         string `json:"name"`
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
		LogoURL:      organization.LogoURL,
		SupportEmail: organization.SupportEmail,
		SupportPhone: organization.SupportPhone,
		TermsURL:     organization.TermsURL,
		PrivacyURL:   organization.PrivacyURL,
		Timezone:     organization.Timezone,
	}
}

func (s *Service) Organization(ctx context.Context) (PublicOrganization, error) {
	organization, err := s.store.Organization(ctx)
	if err != nil {
		return PublicOrganization{}, err
	}

	return publicOrganization(organization), nil
}
