package social

import (
	"time"

	"github.com/google/uuid"

	"vard/internal/model"
)

// providerResponse is a provider as the panel sees it: only whether secrets
// exist, plus the callback address to paste into the provider's console.
type providerResponse struct {
	ID   string           `json:"id"`
	Kind model.SocialKind `json:"kind"`
	Slug string           `json:"slug"`
	Name string           `json:"name"`

	ClientID        string `json:"client_id"`
	HasClientSecret bool   `json:"has_client_secret"`

	TeamID        string `json:"team_id"`
	KeyID         string `json:"key_id"`
	HasPrivateKey bool   `json:"has_private_key"`

	Scopes []string `json:"scopes"`

	// TokenAuth is what the record says, which may be empty; TokenAuthUsed is
	// what that comes to, the kind's own default included.
	TokenAuth     model.SocialTokenAuth `json:"token_auth"`
	TokenAuthUsed model.SocialTokenAuth `json:"token_auth_used"`

	AuthorizeURL string `json:"authorize_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"userinfo_url"`

	IsEnabled          bool `json:"is_enabled"`
	LinkVerifiedEmails bool `json:"link_verified_emails"`
	AllowRegistration  bool `json:"allow_registration"`
	Position           int  `json:"position"`

	// CallbackURL is what the provider has to be told to send people back to.
	CallbackURL string `json:"callback_url"`
	// Identities is how many users sign in with this provider.
	Identities int64     `json:"identities"`
	CreatedAt  time.Time `json:"created_at"`
}

func newProviderResponse(provider model.SocialProvider, issuer string, identities int64) providerResponse {
	scopes := provider.AskedScopes()
	if scopes == nil {
		scopes = []string{}
	}

	return providerResponse{
		ID:                 provider.ID.String(),
		Kind:               provider.Kind,
		Slug:               provider.Slug,
		Name:               provider.Name,
		ClientID:           provider.ClientID,
		HasClientSecret:    len(provider.ClientSecret) > 0,
		TeamID:             provider.TeamID,
		KeyID:              provider.KeyID,
		HasPrivateKey:      len(provider.PrivateKey) > 0,
		Scopes:             scopes,
		TokenAuth:          provider.TokenAuth,
		TokenAuthUsed:      provider.TokenAuthMethod(),
		AuthorizeURL:       provider.AuthorizeURL,
		TokenURL:           provider.TokenURL,
		UserInfoURL:        provider.UserinfoURL,
		IsEnabled:          provider.IsEnabled,
		LinkVerifiedEmails: provider.LinkVerifiedEmails,
		AllowRegistration:  provider.AllowRegistration,
		Position:           provider.Position,
		CallbackURL:        provider.CallbackURL(issuer),
		Identities:         identities,
		CreatedAt:          provider.CreatedAt,
	}
}

// listResponse is every provider plus the kinds the form offers, with their
// endpoints and registration details.
type listResponse struct {
	Providers []providerResponse `json:"providers"`
	Kinds     []model.SocialSpec `json:"kinds"`
}

func newListResponse(providers []model.SocialProvider, counts map[uuid.UUID]int64, issuer string) listResponse {
	out := make([]providerResponse, 0, len(providers))
	for _, provider := range providers {
		out = append(out, newProviderResponse(provider, issuer, counts[provider.ID]))
	}

	return listResponse{Providers: out, Kinds: model.SocialSpecs}
}
