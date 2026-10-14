package model

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SocialProvider struct {
	Base

	Kind SocialKind `gorm:"type:varchar(32);not null" json:"kind"`

	// Slug is part of the redirect URI registered with the provider
	// (/oauth2/social/<slug>/callback), so it never changes.
	Slug string `gorm:"size:64;not null;uniqueIndex" json:"slug"`

	Name string `gorm:"size:100;not null" json:"name"`

	ClientID string `gorm:"size:255;not null" json:"client_id"`

	ClientSecret []byte `gorm:"type:bytea" json:"-"`

	Scopes StringList `gorm:"serializer:json" json:"scopes"`

	IsEnabled bool `gorm:"not null" json:"is_enabled"`

	Position int `gorm:"not null" json:"position"`

	LinkVerifiedEmails bool `gorm:"not null" json:"link_verified_emails"`

	AllowRegistration bool `gorm:"not null" json:"allow_registration"`

	AuthorizeURL string `gorm:"size:512;not null;default:''" json:"authorize_url"`
	TokenURL     string `gorm:"size:512;not null;default:''" json:"token_url"`
	UserinfoURL  string `gorm:"size:512;not null;default:''" json:"userinfo_url"`

	TokenAuth SocialTokenAuth `gorm:"type:varchar(32);not null;default:''" json:"token_auth"`

	// Apple's client secret is a JWT signed per request with the team's key;
	// PrivateKey is the .p8 contents, sealed.
	TeamID     string `gorm:"size:64;not null;default:''" json:"team_id"`
	KeyID      string `gorm:"size:64;not null;default:''" json:"key_id"`
	PrivateKey []byte `gorm:"type:bytea" json:"-"`
}

func (SocialProvider) TableName() string {
	return "social_providers"
}

type StringList []string

// SocialIdentity links a provider's subject to a user. The subject is the only
// reliable identifier: addresses change hands, and some providers give none.
type SocialIdentity struct {
	Base

	UserID     uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ProviderID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_social_identities_subject,priority:1" json:"provider_id"`
	Subject    string    `gorm:"size:255;not null;uniqueIndex:idx_social_identities_subject,priority:2" json:"subject"`

	Email       string     `gorm:"size:255;not null;default:''" json:"email"`
	LastLoginAt *time.Time `json:"last_login_at"`

	Provider *SocialProvider `gorm:"constraint:OnDelete:CASCADE" json:"-"`
	User     *User           `gorm:"constraint:OnDelete:CASCADE" json:"-"`
}

func (SocialIdentity) TableName() string {
	return "social_identities"
}

// SocialAccount is one provider a user signs in with, as their record shows
// it: enough to say which account at the provider it is, and to disconnect it.
type SocialAccount struct {
	// ID is the identity's, which is what disconnecting it names.
	ID       uuid.UUID  `json:"id"`
	Provider string     `json:"provider"`
	Slug     string     `json:"slug"`
	Kind     SocialKind `json:"kind"`

	Email       string     `json:"email"`
	ConnectedAt time.Time  `json:"connected_at"`
	LastLoginAt *time.Time `json:"last_login_at"`
}

// SocialLogin is a sign-in sent to a provider and not yet back, keyed by the
// hashed state. It is a row so it is single use and survives the answer
// arriving in another browser (Apple posts it).
type SocialLogin struct {
	Base

	StateHash  string    `gorm:"size:64;not null;uniqueIndex" json:"-"`
	ProviderID uuid.UUID `gorm:"type:uuid;not null;index" json:"provider_id"`

	Verifier string `gorm:"size:128;not null;default:''" json:"-"`

	Request string `gorm:"size:64;not null;default:''" json:"-"`

	Next string `gorm:"size:512;not null;default:''" json:"-"`

	ExpiresAt time.Time `gorm:"not null;index" json:"-"`
}

func (SocialLogin) TableName() string {
	return "social_logins"
}

const SocialLoginLifetime = 15 * time.Minute

type SocialKind string

const (
	SocialGoogle   SocialKind = "google"
	SocialApple    SocialKind = "apple"
	SocialFacebook SocialKind = "facebook"
	SocialYandex   SocialKind = "yandex"
	SocialVK       SocialKind = "vk"
	SocialOIDC     SocialKind = "oidc"
	SocialOAuth2   SocialKind = "oauth2"
)

type SocialTokenAuth string

const (
	SocialTokenAuthBasic SocialTokenAuth = "basic"
	SocialTokenAuthPost  SocialTokenAuth = "post"
)

var SocialTokenAuths = []SocialTokenAuth{SocialTokenAuthBasic, SocialTokenAuthPost}

type SocialClaims struct {
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	FullName      string `json:"full_name"`
}

// SocialSpec is what is the same for a kind in every installation: endpoints,
// scopes and how to read the answer.
type SocialSpec struct {
	Kind SocialKind `json:"kind"`
	// Label is what the panel calls it, and the name a new provider is given.
	Label string `json:"label"`

	// The endpoints. Empty in a kind whose provider is not known in advance,
	// where the record carries them instead.
	AuthorizeURL string `json:"authorize_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"userinfo_url"`

	// Scopes is what to ask for unless the record says otherwise.
	Scopes []string `json:"scopes"`

	Claims SocialClaims `json:"-"`

	// Custom says the endpoints come from the record, and the panel asks for
	// them.
	Custom bool `json:"custom"`

	IdentityInIDToken bool `json:"-"`

	IDTokenIssuer string `json:"-"`

	EmailInTokenResponse bool `json:"-"`

	UserInfoScheme string `json:"-"`

	AuthorizeParams map[string]string `json:"-"`

	UserInfoParams map[string]string `json:"-"`

	UserInfoTokenParam string `json:"-"`

	PKCE bool `json:"-"`

	// SignedSecret says the client secret is a JWT this server signs with a
	// registered key, rather than a secret the provider gave out (Apple).
	SignedSecret bool `json:"signed_secret"`

	// TokenAuth is how this kind's provider wants the secret presented. A
	// record of a custom kind may say otherwise.
	TokenAuth SocialTokenAuth `json:"token_auth"`

	AssumeEmailVerified bool `json:"-"`

	// Docs is where an administrator registers this server with the provider.
	Docs string `json:"docs"`
}

var SocialSpecs = []SocialSpec{
	{
		Kind:         SocialGoogle,
		Label:        "Google",
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:       []string{"openid", "email", "profile"},
		Claims: SocialClaims{
			Subject:       "sub",
			Email:         "email",
			EmailVerified: "email_verified",
			FirstName:     "given_name",
			LastName:      "family_name",
			FullName:      "name",
		},
		UserInfoScheme: "Bearer",
		PKCE:           true,
		TokenAuth:      SocialTokenAuthPost,
		Docs:           "https://console.cloud.google.com/apis/credentials",
	},
	{
		Kind:  SocialApple,
		Label: "Apple",
		// Apple has no userinfo endpoint: everything it will say about the
		// person is in the id_token, and the name only the first time.
		AuthorizeURL: "https://appleid.apple.com/auth/authorize",
		TokenURL:     "https://appleid.apple.com/auth/token",
		Scopes:       []string{"name", "email"},
		Claims: SocialClaims{
			Subject:       "sub",
			Email:         "email",
			EmailVerified: "email_verified",
		},
		IdentityInIDToken: true,
		IDTokenIssuer:     "https://appleid.apple.com",
		// Asking for a name or an address makes Apple post the answer back
		// instead of redirecting with it in the query.
		AuthorizeParams: map[string]string{"response_mode": "form_post"},
		PKCE:            true,
		SignedSecret:    true,
		TokenAuth:       SocialTokenAuthPost,
		Docs:            "https://developer.apple.com/account/resources/identifiers",
	},
	{
		Kind:         SocialFacebook,
		Label:        "Facebook",
		AuthorizeURL: "https://www.facebook.com/v21.0/dialog/oauth",
		TokenURL:     "https://graph.facebook.com/v21.0/oauth/access_token",
		UserInfoURL:  "https://graph.facebook.com/v21.0/me?fields=id,email,first_name,last_name,name",
		Scopes:       []string{"email", "public_profile"},
		Claims: SocialClaims{
			Subject:   "id",
			Email:     "email",
			FirstName: "first_name",
			LastName:  "last_name",
			FullName:  "name",
		},
		UserInfoScheme: "Bearer",
		PKCE:           true,
		// Facebook hands out an address only once it has confirmed it, and
		// says nothing about it in the answer.
		AssumeEmailVerified: true,
		TokenAuth:           SocialTokenAuthPost,
		Docs:                "https://developers.facebook.com/apps",
	},
	{
		Kind:         SocialYandex,
		Label:        "Yandex ID",
		AuthorizeURL: "https://oauth.yandex.ru/authorize",
		TokenURL:     "https://oauth.yandex.ru/token",
		UserInfoURL:  "https://login.yandex.ru/info?format=json",
		Scopes:       []string{"login:email", "login:info"},
		Claims: SocialClaims{
			Subject:   "id",
			Email:     "default_email",
			FirstName: "first_name",
			LastName:  "last_name",
			FullName:  "real_name",
		},
		UserInfoScheme: "OAuth",
		PKCE:           true,
		// Yandex does not say the address was confirmed, so it can create an
		// unverified account but never link to an existing one.
		TokenAuth: SocialTokenAuthPost,
		Docs:      "https://oauth.yandex.ru/client/new",
	},
	{
		Kind:         SocialVK,
		Label:        "VK ID",
		AuthorizeURL: "https://oauth.vk.com/authorize",
		TokenURL:     "https://oauth.vk.com/access_token",
		UserInfoURL:  "https://api.vk.com/method/users.get",
		Scopes:       []string{"email"},
		Claims: SocialClaims{
			Subject:   "response.0.id",
			FirstName: "response.0.first_name",
			LastName:  "response.0.last_name",
		},
		UserInfoParams:       map[string]string{"v": "5.131"},
		UserInfoTokenParam:   "access_token",
		EmailInTokenResponse: true,
		AuthorizeParams:      map[string]string{"v": "5.131"},
		// VK gives no proof the address was confirmed, so it can create an
		// unverified account but never link to an existing one.
		TokenAuth: SocialTokenAuthPost,
		Docs:      "https://vk.com/editapp?act=create",
	},
	{
		Kind:  SocialOIDC,
		Label: "OpenID Connect",
		Scopes: []string{
			ScopeOpenID, ScopeEmail, ScopeProfile,
		},
		Claims: SocialClaims{
			Subject:       "sub",
			Email:         "email",
			EmailVerified: "email_verified",
			FirstName:     "given_name",
			LastName:      "family_name",
			FullName:      "name",
		},
		Custom:         true,
		UserInfoScheme: "Bearer",
		PKCE:           true,
		TokenAuth:      SocialTokenAuthBasic,
		Docs:           "https://openid.net/developers/how-connect-works/",
	},
	{
		Kind:   SocialOAuth2,
		Label:  "OAuth 2.0",
		Scopes: []string{},
		Claims: SocialClaims{
			Subject:   "id",
			Email:     "email",
			FirstName: "first_name",
			LastName:  "last_name",
			FullName:  "name",
		},
		Custom:         true,
		UserInfoScheme: "Bearer",
		PKCE:           true,
		TokenAuth:      SocialTokenAuthPost,
		Docs:           "https://datatracker.ietf.org/doc/html/rfc6749",
	},
}

func SocialSpecFor(kind SocialKind) (SocialSpec, bool) {
	for _, spec := range SocialSpecs {
		if spec.Kind == kind {
			return spec, true
		}
	}

	return SocialSpec{}, false
}

func (p SocialProvider) Spec() SocialSpec {
	spec, _ := SocialSpecFor(p.Kind)
	return spec
}

func (p SocialProvider) Endpoints() (authorize, token, userInfo string) {
	spec := p.Spec()
	if spec.Custom {
		return p.AuthorizeURL, p.TokenURL, p.UserinfoURL
	}

	return spec.AuthorizeURL, spec.TokenURL, spec.UserInfoURL
}

func (p SocialProvider) TokenAuthMethod() SocialTokenAuth {
	if p.TokenAuth != "" {
		return p.TokenAuth
	}

	if auth := p.Spec().TokenAuth; auth != "" {
		return auth
	}

	return SocialTokenAuthBasic
}

func (p SocialProvider) AskedScopes() []string {
	if len(p.Scopes) > 0 {
		return p.Scopes
	}

	return p.Spec().Scopes
}

func (p SocialProvider) CallbackURL(issuer string) string {
	return strings.TrimRight(issuer, "/") + "/oauth2/social/" + p.Slug + "/callback"
}

var socialSlugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func (p SocialProvider) Validate() error {
	spec, known := SocialSpecFor(p.Kind)
	if !known {
		return fmt.Errorf("kind must be one of: %s", strings.Join(socialKindNames(), ", "))
	}

	switch {
	case p.Slug == "":
		return fmt.Errorf("slug is required")
	case len(p.Slug) > 64 || !socialSlugPattern.MatchString(p.Slug):
		return fmt.Errorf("slug must be lower case letters, numbers and dashes, such as google-workspace")
	}

	switch {
	case strings.TrimSpace(p.Name) == "":
		return fmt.Errorf("name is required")
	case len(p.Name) > 100:
		return fmt.Errorf("name must be at most 100 characters")
	}

	switch {
	case strings.TrimSpace(p.ClientID) == "":
		return fmt.Errorf("client_id is required")
	case len(p.ClientID) > 255:
		return fmt.Errorf("client_id must be at most 255 characters")
	}

	for _, field := range []struct{ name, value string }{
		{"team_id", p.TeamID},
		{"key_id", p.KeyID},
	} {
		if len(field.value) > 64 {
			return fmt.Errorf("%s must be at most 64 characters", field.name)
		}
	}

	if spec.SignedSecret {
		switch {
		case strings.TrimSpace(p.TeamID) == "":
			return fmt.Errorf("team_id is required for %s", spec.Label)
		case strings.TrimSpace(p.KeyID) == "":
			return fmt.Errorf("key_id is required for %s", spec.Label)
		case len(p.PrivateKey) == 0:
			return fmt.Errorf("private_key is required for %s: the .p8 file from the developer account", spec.Label)
		}
	} else if len(p.ClientSecret) == 0 {
		return fmt.Errorf("client_secret is required")
	}

	if spec.Custom {
		// The authorize URL is only followed by the browser, so http is
		// allowed; token and userinfo URLs carry secrets and must be https,
		// except on localhost.
		if err := socialEndpoint("authorize_url", p.AuthorizeURL, true, false); err != nil {
			return err
		}
		if err := socialEndpoint("token_url", p.TokenURL, true, true); err != nil {
			return err
		}

		if err := socialEndpoint("userinfo_url", p.UserinfoURL, p.Kind == SocialOAuth2, true); err != nil {
			return err
		}
	}

	if p.TokenAuth != "" && !slices.Contains(SocialTokenAuths, p.TokenAuth) {
		return fmt.Errorf("token_auth must be basic or post")
	}

	for _, scope := range p.Scopes {
		if strings.ContainsAny(scope, " \t\r\n") {
			return fmt.Errorf("scopes: %q must not contain spaces — list them one at a time", scope)
		}
	}

	return nil
}

func socialEndpoint(field, value string, required, secure bool) error {
	if value == "" {
		if required {
			return fmt.Errorf("%s is required for this kind of provider", field)
		}
		return nil
	}

	if len(value) > 512 {
		return fmt.Errorf("%s must be at most 512 characters", field)
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be a full address starting with http:// or https://", field)
	}

	if secure && parsed.Scheme == "http" && !IsLocalhost(parsed) {
		return fmt.Errorf("%s must use https", field)
	}

	return nil
}

func socialKindNames() []string {
	names := make([]string, 0, len(SocialSpecs))
	for _, spec := range SocialSpecs {
		names = append(names, string(spec.Kind))
	}

	return names
}
