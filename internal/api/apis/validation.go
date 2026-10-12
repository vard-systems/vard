package apis

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/model"
)

func (r *apiRequest) applyTo(api *model.API, creating bool) error {
	r.clean()

	if err := validate.Struct(r); err != nil {
		return err
	}

	if len(r.Scopes) > maxScopes {
		return badRequest(fmt.Sprintf("an API may have at most %d scopes", maxScopes))
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
				return badRequest("scopes: " + sent.Name + " refers to a scope this API does not have")
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
		return badRequest(err.Error())
	}

	return nil
}

func badRequest(message string) error {
	return respond.Fault{Status: http.StatusBadRequest, Message: message}
}
