package social

const targetType = "social_provider"

// providerRequest is a PATCH: a setting left out stays as it is, so
// `{"enabled": false}` cannot empty the rest. Secrets are never sent back, so
// only a new value replaces one.
type providerRequest struct {
	// Kind and Slug are read only on registration; they decide the registered
	// callback address.
	Kind string `json:"kind"`
	Slug string `json:"slug"`

	Name     *string `json:"name"`
	ClientID *string `json:"client_id"`

	ClientSecret *string `json:"client_secret"`
	PrivateKey   *string `json:"private_key"`

	TeamID *string `json:"team_id"`
	KeyID  *string `json:"key_id"`

	Scopes *[]string `json:"scopes"`

	// TokenAuth is how the secret is presented at the token endpoint. An
	// empty string leaves it to the kind's own default.
	TokenAuth *string `json:"token_auth"`

	AuthorizeURL *string `json:"authorize_url"`
	TokenURL     *string `json:"token_url"`
	UserInfoURL  *string `json:"userinfo_url"`

	IsEnabled          *bool `json:"is_enabled"`
	LinkVerifiedEmails *bool `json:"link_verified_emails"`
	AllowRegistration  *bool `json:"allow_registration"`

	// Position is where the button sits. Left out, a new provider goes last
	// and an existing one stays where it is.
	Position *int `json:"position"`
}
