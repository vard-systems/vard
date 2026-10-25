package oidc

import (
	"context"

	"vard/internal/model"
)

// PublicApplication is what anyone may know about an application: what its
// sign-in pages show.
type PublicApplication struct {
	Name              string `json:"name"`
	LogoURL           string `json:"logo_url"`
	WebsiteURL        string `json:"website_url"`
	PrivacyURL        string `json:"privacy_url"`
	TermsURL          string `json:"terms_url"`
	AllowRegistration bool   `json:"allow_registration"`
}

func NewPublicApplication(app *model.Application) PublicApplication {
	return PublicApplication{
		Name:              app.Name,
		LogoURL:           app.LogoURL,
		WebsiteURL:        app.WebsiteURL,
		PrivacyURL:        app.PrivacyURL,
		TermsURL:          app.TermsURL,
		AllowRegistration: app.AllowRegistration && app.IsEnabled,
	}
}

func (s *Service) ApplicationByClientID(ctx context.Context, clientID string) (*PublicApplication, error) {
	app, err := s.store.ApplicationByClientID(ctx, clientID)
	if err != nil {
		return nil, err
	}

	public := NewPublicApplication(app)
	return &public, nil
}
