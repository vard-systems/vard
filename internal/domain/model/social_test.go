package model

import (
	"slices"
	"testing"
)

func TestSocialProviderValidate(t *testing.T) {
	google := func() SocialProvider {
		p := DefaultSocialProvider(SocialGoogle)
		p.ClientID = "1234.apps.googleusercontent.com"
		p.ClientSecret = []byte("sealed")
		return p
	}

	apple := func() SocialProvider {
		p := DefaultSocialProvider(SocialApple)
		p.ClientID = "com.example.service"
		p.TeamID, p.KeyID, p.PrivateKey = "TEAM123456", "KEY1234567", []byte("sealed .p8")
		return p
	}

	custom := func() SocialProvider {
		p := DefaultSocialProvider(SocialOIDC)
		p.Slug, p.Name = "keycloak", "Keycloak"
		p.ClientID, p.ClientSecret = "keycloak", []byte("sealed")
		p.AuthorizeURL = "https://sso.example.com/authorize"
		p.TokenURL = "https://sso.example.com/token"
		p.UserinfoURL = "https://sso.example.com/userinfo"
		return p
	}

	tests := []struct {
		name     string
		provider SocialProvider
		want     string // the message, or "" when it is fine
	}{
		{name: "a provider with a built-in kind", provider: google()},
		{name: "apple, which signs its secret", provider: apple()},
		{name: "a provider whose endpoints are its own", provider: custom()},
		{
			name:     "a kind nobody has",
			provider: func() SocialProvider { p := google(); p.Kind = "myspace"; return p }(),
			want:     "kind must be one of: google, apple, facebook, yandex, vk, oidc, oauth2",
		},
		{
			name:     "a slug with a space",
			provider: func() SocialProvider { p := google(); p.Slug = "google workspace"; return p }(),
			want:     "slug must be lower case letters, numbers and dashes, such as google-workspace",
		},
		{
			name:     "no name to put on the button",
			provider: func() SocialProvider { p := google(); p.Name = " "; return p }(),
			want:     "name is required",
		},
		{
			name:     "no client id",
			provider: func() SocialProvider { p := google(); p.ClientID = ""; return p }(),
			want:     "client_id is required",
		},
		{
			name:     "no secret, for a kind that needs one",
			provider: func() SocialProvider { p := google(); p.ClientSecret = nil; return p }(),
			want:     "client_secret is required",
		},
		{
			name:     "apple without its key",
			provider: func() SocialProvider { p := apple(); p.PrivateKey = nil; return p }(),
			want:     "private_key is required for Apple: the .p8 file from the developer account",
		},
		{
			name:     "apple without a team",
			provider: func() SocialProvider { p := apple(); p.TeamID = ""; return p }(),
			want:     "team_id is required for Apple",
		},
		{
			name:     "a custom provider with nowhere to send people",
			provider: func() SocialProvider { p := custom(); p.AuthorizeURL = ""; return p }(),
			want:     "authorize_url is required for this kind of provider",
		},
		{
			name:     "a custom provider whose endpoint is not an address",
			provider: func() SocialProvider { p := custom(); p.TokenURL = "sso.example.com/token"; return p }(),
			want:     "token_url must be a full address starting with http:// or https://",
		},
		{
			name: "an OAuth 2.0 provider with no profile to read",
			provider: func() SocialProvider {
				p := custom()
				p.Kind, p.UserinfoURL = SocialOAuth2, ""
				return p
			}(),
			want: "userinfo_url is required for this kind of provider",
		},
		{
			name:     "OpenID Connect may leave the profile out, since the id_token carries one",
			provider: func() SocialProvider { p := custom(); p.UserinfoURL = ""; return p }(),
		},
		{
			name:     "scopes listed as one string",
			provider: func() SocialProvider { p := google(); p.Scopes = StringList{"openid email"}; return p }(),
			want:     `scopes: "openid email" must not contain spaces — list them one at a time`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.Validate()

			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nothing", err)
			case tt.want != "" && err == nil:
				t.Fatalf("Validate() = nothing, want %q", tt.want)
			case tt.want != "" && err.Error() != tt.want:
				t.Errorf("Validate() = %q, want %q", err, tt.want)
			}
		})
	}
}

// Every kind a panel can offer has to be usable: one this server knows the
// endpoints of, or one it asks the administrator for them.
func TestSocialSpecsAreComplete(t *testing.T) {
	for _, spec := range SocialSpecs {
		t.Run(string(spec.Kind), func(t *testing.T) {
			if spec.Label == "" || spec.Docs == "" {
				t.Errorf("%s has no label or no link to where it is registered", spec.Kind)
			}

			if spec.Custom {
				if spec.AuthorizeURL != "" || spec.TokenURL != "" {
					t.Errorf("%s asks for endpoints and also has its own", spec.Kind)
				}
			} else {
				if spec.AuthorizeURL == "" || spec.TokenURL == "" {
					t.Errorf("%s has no endpoints and does not ask for any", spec.Kind)
				}
				if !spec.IdentityInIDToken && spec.UserInfoURL == "" {
					t.Errorf("%s has nowhere to read the identity from", spec.Kind)
				}
			}

			if spec.Claims.Subject == "" {
				t.Errorf("%s does not say where the subject is", spec.Kind)
			}

			// A provider of this kind, filled in the way the panel fills one,
			// has to be valid once its credentials are typed in.
			p := DefaultSocialProvider(spec.Kind)
			p.ClientID, p.ClientSecret = "id", []byte("secret")
			p.TeamID, p.KeyID, p.PrivateKey = "team", "key", []byte("pem")
			p.AuthorizeURL = "https://example.com/authorize"
			p.TokenURL = "https://example.com/token"
			p.UserinfoURL = "https://example.com/userinfo"

			if err := p.Validate(); err != nil {
				t.Errorf("a new %s provider is not valid: %v", spec.Kind, err)
			}
		})
	}
}

func TestSocialProviderCallbackURL(t *testing.T) {
	p := DefaultSocialProvider(SocialGoogle)

	if got := p.CallbackURL("https://id.example.com/"); got != "https://id.example.com/oauth2/social/google/callback" {
		t.Errorf("CallbackURL() = %q", got)
	}
}

// DefaultSocialProvider is a provider of a kind with the usual settings.
func DefaultSocialProvider(kind SocialKind) SocialProvider {
	spec, _ := SocialSpecFor(kind)

	return SocialProvider{
		Kind:               kind,
		Slug:               string(kind),
		Name:               spec.Label,
		Scopes:             slices.Clone(spec.Scopes),
		IsEnabled:          true,
		LinkVerifiedEmails: true,
		AllowRegistration:  true,
	}
}

// Only a provider that says an address is verified, and is allowed to, takes
// over the existing account with that address. Facebook's verification is
// assumed, never said, so it never does, whatever the panel's setting.
func TestOnlyAProviderThatSaysAnAddressIsVerifiedLinksByIt(t *testing.T) {
	tests := []struct {
		name string
		kind SocialKind
		link bool
		want bool
	}{
		{"Google, allowed to link", SocialGoogle, true, true},
		{"Google, not allowed to", SocialGoogle, false, false},
		{"a provider of the administrator's own, allowed to link", SocialOIDC, true, true},
		{"Facebook, allowed to link", SocialFacebook, true, false},
		{"Facebook, not allowed to", SocialFacebook, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := SocialProvider{Kind: tt.kind, LinkVerifiedEmails: tt.link}
			if got := p.LinksByAddress(); got != tt.want {
				t.Errorf("LinksByAddress() = %v, want %v", got, tt.want)
			}
		})
	}

	// Every kind that assumes verification is covered, not just Facebook.
	for _, spec := range SocialSpecs {
		if !spec.AssumeEmailVerified {
			continue
		}
		if (SocialProvider{Kind: spec.Kind, LinkVerifiedEmails: true}).LinksByAddress() {
			t.Errorf("%s assumes addresses are verified, and still links by them", spec.Kind)
		}
		if (SocialProvider{Kind: spec.Kind}).ProvesAddresses() {
			t.Errorf("%s assumes addresses are verified, and still makes accounts verified", spec.Kind)
		}
	}
	if !(SocialProvider{Kind: SocialGoogle}).ProvesAddresses() {
		t.Error("Google says itself that an address is verified, and its accounts start unverified")
	}
}
