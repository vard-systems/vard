package organization

import (
	"encoding/base64"
	"errors"
	"slices"
	"time"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/validate"
)

func (r *organizationRequest) applyTo(organization *model.Organization) ([]string, error) {
	updated := *organization
	updated.Name = validate.Text(r.Name, organization.Name)
	updated.Slug = validate.Lower(r.Slug, organization.Slug)
	updated.SupportEmail = validate.Text(r.SupportEmail, organization.SupportEmail)
	updated.SupportPhone = validate.Text(r.SupportPhone, organization.SupportPhone)
	updated.TermsURL = validate.Text(r.TermsURL, organization.TermsURL)
	updated.PrivacyURL = validate.Text(r.PrivacyURL, organization.PrivacyURL)
	updated.Timezone = validate.Text(r.Timezone, organization.Timezone)

	if err := updated.Validate(); err != nil {
		return nil, respond.Invalid(err.Error())
	}

	changed := []string{}
	for _, field := range []struct {
		name   string
		was    string
		became string
	}{
		{"name", organization.Name, updated.Name},
		{"slug", organization.Slug, updated.Slug},
		{"support_email", organization.SupportEmail, updated.SupportEmail},
		{"support_phone", organization.SupportPhone, updated.SupportPhone},
		{"terms_url", organization.TermsURL, updated.TermsURL},
		{"privacy_url", organization.PrivacyURL, updated.PrivacyURL},
		{"timezone", organization.Timezone, updated.Timezone},
	} {
		if field.was != field.became {
			changed = append(changed, field.name)
		}
	}

	// Users who accepted the old links are asked again by the Agreements step.
	if slices.Contains(changed, "terms_url") || slices.Contains(changed, "privacy_url") {
		now := time.Now()
		updated.AgreementsUpdatedAt = &now
	}

	*organization = updated

	return changed, nil
}

// logo is the logo in the request, checked by what its bytes are.
func (r *logoRequest) logo() (model.OrganizationLogo, error) {
	data, err := base64.StdEncoding.DecodeString(r.Data)
	if err != nil || len(data) == 0 {
		return model.OrganizationLogo{}, respond.InvalidBody.Fault(nil)
	}

	logo, err := model.NewOrganizationLogo(data)
	switch {
	case errors.Is(err, model.ErrLogoTooLarge):
		return model.OrganizationLogo{}, logoTooLarge.Fault(nil)
	case errors.Is(err, model.ErrLogoType):
		return model.OrganizationLogo{}, logoType.Fault(nil)
	case errors.Is(err, model.ErrLogoDimension):
		return model.OrganizationLogo{}, logoDimension.Fault(nil)
	case err != nil:
		return model.OrganizationLogo{}, err
	}

	return logo, nil
}
