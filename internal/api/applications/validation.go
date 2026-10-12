package applications

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

func (r *applicationRequest) applyTo(app *model.Application, creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	if len(r.RedirectURIs) > maxURIs || len(r.PostLogoutRedirectURIs) > maxURIs {
		return badRequest(fmt.Sprintf("an application may have at most %d redirect URIs of each kind", maxURIs))
	}

	if creating && !model.ApplicationType(r.Type).Valid() {
		return badRequest("type must be one of: web, spa, native, m2m")
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

	app.Normalise()

	if !app.HasSecret() {
		app.ClientSecretHash = ""
		app.SecretHint = ""
		app.SecretCreatedAt = nil
	}

	if err := app.Validate(); err != nil {
		return badRequest(err.Error())
	}

	return nil
}

func badRequest(message string) error {
	return respond.Fault{Status: http.StatusBadRequest, Message: message}
}

func flowID(sent string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(sent)
	if trimmed == "" {
		return nil, nil
	}

	id, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, badRequest("login_flow_id must be the id of a login flow")
	}

	return &id, nil
}
