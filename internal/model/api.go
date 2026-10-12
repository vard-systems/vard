package model

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"loginer/internal/brand"
)

type API struct {
	Base

	Name        string `gorm:"size:100;not null" json:"name"`
	Identifier  string `gorm:"size:255;not null;uniqueIndex" json:"identifier"`
	Description string `gorm:"size:255" json:"description"`

	EnforceRoles bool `gorm:"not null" json:"enforce_roles"`

	// SigningAlgorithm is asymmetric, so the API verifies with published keys
	// and never holds one that signs.
	SigningAlgorithm string `gorm:"type:varchar(16);not null" json:"signing_algorithm"`

	TokenLifetime int `gorm:"not null" json:"token_lifetime"`

	AllowOfflineAccess bool `gorm:"not null" json:"allow_offline_access"`

	System string `gorm:"type:varchar(16);not null;default:''" json:"system,omitempty"`

	Scopes []APIScope `gorm:"constraint:OnDelete:CASCADE" json:"scopes,omitempty"`
}

func (API) TableName() string {
	return "apis"
}

const (
	SystemAdminAPI   = "admin"
	SystemAccountAPI = "account"
)

const (
	ScopeAccountRead  = "account.read"
	ScopeAccountWrite = "account.write"
)

const AdminCLIClientID = "admin-cli"

func SystemAPIs() []API {
	admin := API{
		Name:             "Admin API",
		Identifier:       brand.AdminAPIIdentifier,
		Description:      "This server's admin API, for software managing users and settings as a service.",
		System:           SystemAdminAPI,
		SigningAlgorithm: AlgRS256,
	}
	for _, permission := range AdminPermissions {
		admin.Scopes = append(admin.Scopes, APIScope{
			Name:        permission.Name,
			Description: permission.Description,
			IsDefault:   true,
		})
	}

	account := API{
		Name:             "Account API",
		Identifier:       brand.AccountAPIIdentifier,
		Description:      "A signed-in user's own account: what the sign-in app's account pages do.",
		System:           SystemAccountAPI,
		SigningAlgorithm: AlgRS256,
		Scopes: []APIScope{
			{Name: ScopeAccountRead, Description: "Read the user's profile, sessions and connected applications", IsDefault: true},
			{Name: ScopeAccountWrite, Description: "Change the user's profile, password and email, and end their sessions"},
		},
	}

	return []API{admin, account}
}

// APIScope is one thing a token for an API may allow — "orders:read" — which
// the API checks for.
type APIScope struct {
	Base

	APIID       uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_api_scopes_api_name,priority:1" json:"api_id"`
	Name        string    `gorm:"size:128;not null;uniqueIndex:idx_api_scopes_api_name,priority:2" json:"name"`
	Description string    `gorm:"size:255" json:"description"`

	// IsDefault adds the scope to every token for the API, subject to the same
	// allowance and role rules.
	IsDefault bool `gorm:"not null" json:"is_default"`
}

// Signing algorithms for access tokens. HS256 is excluded: every API would have
// to hold the signing secret.
const (
	AlgRS256 = "RS256"
	AlgPS256 = "PS256"
	AlgES256 = "ES256"
)

var SigningAlgorithms = []string{AlgRS256, AlgPS256, AlgES256}

const (
	minAPITokenLifetime = 60
	maxAPITokenLifetime = 24 * 60 * 60
)

func (APIScope) TableName() string {
	return "api_scopes"
}

type ApplicationAPI struct {
	ApplicationID uuid.UUID    `gorm:"type:uuid;primaryKey" json:"application_id"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	APIID         uuid.UUID    `gorm:"type:uuid;primaryKey" json:"api_id"`
	API           *API         `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	CreatedAt     time.Time    `json:"created_at"`
}

func (ApplicationAPI) TableName() string {
	return "application_apis"
}

type ApplicationAPIScope struct {
	ApplicationID uuid.UUID    `gorm:"type:uuid;primaryKey" json:"application_id"`
	Application   *Application `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	APIScopeID    uuid.UUID    `gorm:"type:uuid;primaryKey" json:"api_scope_id"`
	APIScope      *APIScope    `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

func (ApplicationAPIScope) TableName() string {
	return "application_api_scopes"
}

var scopeNamePattern = regexp.MustCompile(`^[a-z][a-z0-9]*([:._-][a-z0-9]+)*$`)

func (a API) Validate() error {
	if strings.TrimSpace(a.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if a.Identifier == "" {
		return fmt.Errorf("identifier is required")
	}
	if strings.ContainsAny(a.Identifier, " \t\r\n") {
		return fmt.Errorf("identifier must not contain spaces")
	}
	if len(a.Identifier) > 255 {
		return fmt.Errorf("identifier must be at most 255 characters")
	}

	if !slices.Contains(SigningAlgorithms, a.SigningAlgorithm) {
		return fmt.Errorf("signing_algorithm must be one of: %s", strings.Join(SigningAlgorithms, ", "))
	}

	if a.TokenLifetime != 0 && (a.TokenLifetime < minAPITokenLifetime || a.TokenLifetime > maxAPITokenLifetime) {
		return fmt.Errorf("token_lifetime must be between %d and %d seconds, or 0 to use the application's", minAPITokenLifetime, maxAPITokenLifetime)
	}

	seen := map[string]bool{}
	for _, scope := range a.Scopes {
		switch {
		case !scopeNamePattern.MatchString(scope.Name) || len(scope.Name) > 128:
			return fmt.Errorf("scopes: %q must be lower case words joined by : . - or _, such as orders:read", scope.Name)
		case slices.Contains(Scopes, scope.Name):
			return fmt.Errorf("scopes: %q is an OpenID Connect scope and cannot be an API scope", scope.Name)
		case seen[scope.Name]:
			return fmt.Errorf("scopes: %q is listed twice", scope.Name)
		case len(scope.Description) > 255:
			return fmt.Errorf("scopes: the description of %q must be at most 255 characters", scope.Name)
		}
		seen[scope.Name] = true
	}

	return nil
}
