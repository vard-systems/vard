package apis

import (
	"strings"

	"github.com/google/uuid"

	"vard/core/model"
)

// maxLogEntries caps how many log entries one request returns.
const maxLogEntries = 100

const targetType = "api"

// maxScopes caps how many scopes one API may have.
const maxScopes = 200

// apiRequest is the create and update body. Scopes replace the API's: sent with
// an id they are kept, without one they are new, left out they are removed.
type apiRequest struct {
	Name        string `json:"name" validate:"required,max=100"`
	Identifier  string `json:"identifier" validate:"max=255"`
	Description string `json:"description" validate:"max=255"`
	// EnforceRoles and AllowOfflineAccess may be left out: a new API then
	// enforces roles and refuses refresh tokens; an existing one keeps them.
	EnforceRoles *bool `json:"enforce_roles"`

	// SigningAlgorithm left empty is RS256. TokenLifetime is in seconds, and
	// zero leaves it to each application.
	SigningAlgorithm   string `json:"signing_algorithm"`
	TokenLifetime      int    `json:"token_lifetime"`
	AllowOfflineAccess *bool  `json:"allow_offline_access"`

	Scopes []scopeRequest `json:"scopes" validate:"dive"`
}

type scopeRequest struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default"`
}

func (r *apiRequest) clean() {
	r.Name = strings.TrimSpace(r.Name)
	r.Identifier = strings.TrimSpace(r.Identifier)
	r.Description = strings.TrimSpace(r.Description)
	r.SigningAlgorithm = strings.ToUpper(strings.TrimSpace(r.SigningAlgorithm))
	if r.SigningAlgorithm == "" {
		r.SigningAlgorithm = model.AlgRS256
	}

	for i := range r.Scopes {
		r.Scopes[i].Name = strings.ToLower(strings.TrimSpace(r.Scopes[i].Name))
		r.Scopes[i].Description = strings.TrimSpace(r.Scopes[i].Description)
	}
}

// maxIntrospectors caps how many applications one API may list.
const maxIntrospectors = 50

// introspectorsRequest names every application that may introspect the API's
// tokens; one left out no longer may.
type introspectorsRequest struct {
	ApplicationIDs []uuid.UUID `json:"application_ids"`
}
