package applications

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

var (
	typeFixed              = respond.Define(http.StatusBadRequest, "application_type_fixed", respond.Admin)
	noSecret               = respond.Define(http.StatusBadRequest, "application_no_secret", respond.Admin)
	foreignScope           = respond.Define(http.StatusBadRequest, "application_api_scope_foreign", respond.Admin)
	previewUserUnknown     = respond.Define(http.StatusBadRequest, "token_preview_user_unknown", respond.Admin)
	previewAudienceUnknown = respond.Define(http.StatusBadRequest, "token_preview_audience_unknown", respond.Admin)
	tooManyURIs            = respond.Define(http.StatusBadRequest, "application_too_many_uris", respond.Admin)
	typeInvalid            = respond.Define(http.StatusBadRequest, "application_type_invalid", respond.Admin)
	loginFlowUnknown       = respond.Define(http.StatusBadRequest, "application_login_flow_unknown", respond.Admin)
)

func (r *applicationRequest) applyTo(app *model.Application, creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	if len(r.RedirectURIs) > maxURIs || len(r.PostLogoutRedirectURIs) > maxURIs {
		return tooManyURIs.With("max", maxURIs)
	}

	if creating && !model.ApplicationType(r.Type).Valid() {
		return typeInvalid.With()
	}

	app.Name = r.Name
	app.Description = r.Description
	app.LogoURL = r.LogoURL
	app.WebsiteURL = r.WebsiteURL
	app.PrivacyURL = r.PrivacyURL
	app.TermsURL = r.TermsURL
	app.TokenAuthMethod = model.AuthMethod(r.TokenAuthMethod)
	app.GrantTypes = r.GrantTypes
	app.RedirectURIs = r.RedirectURIs
	app.PostLogoutRedirectURIs = r.PostLogoutRedirectURIs
	app.Scopes = r.Scopes
	app.RequirePKCE = validate.Flag(r.RequirePKCE, app.RequirePKCE)
	app.AccessTokenLifetime = r.AccessTokenLifetime
	app.IDTokenLifetime = r.IDTokenLifetime
	app.RefreshTokenLifetime = r.RefreshTokenLifetime
	app.AssertRoles = validate.Flag(r.AssertRoles, app.AssertRoles)
	app.RequireRoleAssignment = validate.Flag(r.RequireRoleAssignment, app.RequireRoleAssignment)
	app.IsEnabled = validate.Flag(r.IsEnabled, app.IsEnabled)
	app.AllowRegistration = validate.Flag(r.AllowRegistration, app.AllowRegistration)

	if r.LoginFlowID != nil {
		flow, err := flowID(*r.LoginFlowID)
		if err != nil {
			return err
		}

		app.LoginFlowID = flow
	}

	app.Normalize()

	if !app.HasSecret() {
		app.ClientSecretHash = ""
		app.SecretHint = ""
		app.SecretCreatedAt = nil
	}

	if err := app.Validate(); err != nil {
		return respond.Invalid(err.Error())
	}

	return nil
}

func flowID(sent string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(sent)
	if trimmed == "" {
		return nil, nil
	}

	id, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, loginFlowUnknown.With()
	}

	return &id, nil
}
