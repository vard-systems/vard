package apis

import (
	"net/http"
	"slices"
	"strings"

	"github.com/google/uuid"

	"vard/internal/domain/model"
	"vard/internal/interfaces/api/kit/respond"
	"vard/internal/interfaces/api/kit/validate"
)

var (
	identifierFixed = respond.Define(http.StatusBadRequest, "api_identifier_fixed", respond.Admin)
	identifierTaken = respond.Define(http.StatusConflict, "api_identifier_taken", respond.Admin)
	tooManyScopes   = respond.Define(http.StatusBadRequest, "api_too_many_scopes", respond.Admin)
	foreignScope    = respond.Define(http.StatusBadRequest, "api_scope_foreign", respond.Admin)

	tooManyIntrospectors = respond.Define(http.StatusBadRequest, "api_too_many_introspectors", respond.Admin)
	publicIntrospector   = respond.Define(http.StatusBadRequest, "api_introspector_public", respond.Admin)
)

// introspectors checks the applications named against the ones there are,
// each once: every one must exist and be confidential, since what it may
// read is the API's users' subject, scope and roles, and a public client is
// anyone who knows its id.
func (r *introspectorsRequest) introspectors(found []model.Application) ([]uuid.UUID, error) {
	ids := slices.Compact(slices.SortedFunc(slices.Values(r.ApplicationIDs), func(a, b uuid.UUID) int {
		return strings.Compare(a.String(), b.String())
	}))
	if len(ids) > maxIntrospectors {
		return nil, tooManyIntrospectors.With("max", maxIntrospectors)
	}

	for _, id := range ids {
		i := slices.IndexFunc(found, func(app model.Application) bool { return app.ID == id })
		if i < 0 {
			return nil, respond.ApplicationNotFound.With()
		}
		if !found[i].Confidential() {
			return nil, publicIntrospector.With("name", found[i].Name)
		}
	}
	return ids, nil
}

func (r *apiRequest) applyTo(api *model.API, creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	if len(r.Scopes) > maxScopes {
		return tooManyScopes.With("max", maxScopes)
	}

	// A scope sent with an id has to be one of this API's own: an id from
	// another API would move that scope, and its grants, over here.
	own := map[string]bool{}
	for _, scope := range api.Scopes {
		own[scope.ID.String()] = true
	}

	scopes := make([]model.APIScope, 0, len(r.Scopes))
	for _, sent := range r.Scopes {
		scope := model.APIScope{Name: sent.Name, Description: sent.Description, IsDefault: sent.IsDefault}

		if sent.ID != uuid.Nil {
			if creating || !own[sent.ID.String()] {
				return foreignScope.With("scope", sent.Name)
			}
			scope.ID = sent.ID
		}

		scopes = append(scopes, scope)
	}

	if creating {
		api.Identifier = r.Identifier
	}
	api.Name = r.Name
	api.Description = r.Description
	api.EnforceRoles = validate.Flag(r.EnforceRoles, api.EnforceRoles)
	api.SigningAlgorithm = r.SigningAlgorithm
	api.TokenLifetime = r.TokenLifetime
	api.AllowOfflineAccess = validate.Flag(r.AllowOfflineAccess, api.AllowOfflineAccess)
	api.Scopes = scopes

	if err := api.Validate(); err != nil {
		return respond.Invalid(err.Error())
	}

	return nil
}
