package social

import (
	"errors"
	"reflect"
	"testing"

	"loginer/internal/api/respond"
	"loginer/internal/jose"
	"loginer/internal/model"
)

func sealer(t *testing.T) *jose.Sealer {
	t.Helper()

	s, err := jose.NewSealer("a secret long enough to derive a key from")
	if err != nil {
		t.Fatal(err)
	}

	return s
}

func TestProviderRequestApplyToCreating(t *testing.T) {
	tests := []struct {
		name    string
		request providerRequest
		want    string // the message, or "" when the request is fine
		check   func(model.SocialProvider) error
	}{
		{
			name: "a provider of a kind this server knows",
			request: providerRequest{
				Kind:         " Google ",
				Name:         new("  Google  "),
				ClientID:     new(" 1234.apps.googleusercontent.com "),
				ClientSecret: new("a secret"),
				Scopes:       &[]string{"openid", " email ", ""},
			},
			check: func(p model.SocialProvider) error {
				switch {
				case p.Kind != model.SocialGoogle:
					return errors.New("kind = " + string(p.Kind))
				case p.Slug != "google":
					return errors.New("slug = " + p.Slug + ", want the kind's own name")
				case p.Name != "Google":
					return errors.New("name = " + p.Name)
				case !reflect.DeepEqual([]string(p.Scopes), []string{"openid", "email"}):
					return errors.New("scopes were not tidied")
				case len(p.ClientSecret) == 0:
					return errors.New("the secret was not sealed")
				case string(p.ClientSecret) == "a secret":
					return errors.New("the secret was stored as it was typed")
				}
				return nil
			},
		},
		{
			name: "a second provider of the same kind, under its own name",
			request: providerRequest{
				Kind: "oidc", Slug: "Staff-SSO", Name: new("Staff SSO"),
				ClientID: new("staff-portal"), ClientSecret: new("a secret"),
				AuthorizeURL: new("https://sso.example.com/authorize"),
				TokenURL:     new("https://sso.example.com/token"),
				UserInfoURL:  new("https://sso.example.com/userinfo"),
			},
			check: func(p model.SocialProvider) error {
				if p.Slug != "staff-sso" {
					return errors.New("slug = " + p.Slug)
				}
				return nil
			},
		},
		{
			name: "a kind nobody offers",
			request: providerRequest{
				Kind: "myspace", Name: new("MySpace"), ClientID: new("id"), ClientSecret: new("s"),
			},
			want: "The kind must be one of: google, apple, facebook, yandex, vk, oidc, oauth2.",
		},
		{
			name:    "no name for the button",
			request: providerRequest{Kind: "google", ClientID: new("id"), ClientSecret: new("s")},
			want:    "name is required",
		},
		{
			name:    "no client id",
			request: providerRequest{Kind: "google", Name: new("Google"), ClientSecret: new("s")},
			want:    "client_id is required",
		},
		{
			name:    "no secret",
			request: providerRequest{Kind: "google", Name: new("Google"), ClientID: new("id")},
			want:    "client_secret is required",
		},
		{
			name: "a custom provider with nowhere to send people",
			request: providerRequest{
				Kind: "oidc", Name: new("Staff SSO"), ClientID: new("id"), ClientSecret: new("s"),
			},
			want: "authorize_url is required for this kind of provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			provider := model.SocialProvider{IsEnabled: true, LinkVerifiedEmails: true, AllowRegistration: true}

			err := request.applyTo(&provider, sealer(t), true)

			if tt.want != "" {
				var fault respond.Fault
				if !errors.As(err, &fault) {
					t.Fatalf("applyTo() = %v, want a Fault", err)
				}
				if fault.Message != tt.want {
					t.Errorf("message = %q, want %q", fault.Message, tt.want)
				}
				return
			}

			if err != nil {
				t.Fatalf("applyTo() = %v, want nothing", err)
			}
			if tt.check != nil {
				if err := tt.check(provider); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

// An update says what it changes and nothing else. A request that mentions
// one setting must leave the rest of the record alone — a caller turning a
// provider off would otherwise empty the endpoints and keys around it.
func TestProviderRequestUpdatesOnlyWhatItMentions(t *testing.T) {
	seal := sealer(t)

	stored, err := seal.SealBytes([]byte("the stored key"))
	if err != nil {
		t.Fatal(err)
	}

	apple := model.SocialProvider{
		Kind: model.SocialApple, Slug: "apple", Name: "Apple",
		ClientID: "com.example.service",
		TeamID:   "TEAM123456", KeyID: "KEY1234567", PrivateKey: stored,
		Scopes:             model.StringList{"name", "email"},
		IsEnabled:          false,
		LinkVerifiedEmails: true,
		AllowRegistration:  true,
	}

	was := apple
	request := providerRequest{IsEnabled: new(true)}

	if err := request.applyTo(&apple, seal, false); err != nil {
		t.Fatalf("applyTo() = %v, want nothing: turning a provider on says nothing about its keys", err)
	}

	switch {
	case !apple.IsEnabled:
		t.Error("the provider was not turned on")
	case apple.Name != was.Name || apple.ClientID != was.ClientID:
		t.Errorf("the name or the client id was emptied: %+v", apple)
	case apple.TeamID != was.TeamID || apple.KeyID != was.KeyID:
		t.Errorf("the team or the key id was emptied: %+v", apple)
	case !reflect.DeepEqual(apple.PrivateKey, was.PrivateKey):
		t.Error("the stored key did not survive")
	case !reflect.DeepEqual(apple.Scopes, was.Scopes):
		t.Errorf("the scopes were emptied: %v", apple.Scopes)
	case !apple.LinkVerifiedEmails || !apple.AllowRegistration:
		t.Error("a switch nobody mentioned was thrown")
	}
}

// The kind and the slug are in the address registered with the provider, so
// an update may not move them.
func TestProviderRequestKeepsKindAndSlug(t *testing.T) {
	seal := sealer(t)

	provider := model.SocialProvider{
		Kind: model.SocialGoogle, Slug: "google", Name: "Google",
		ClientID: "id", ClientSecret: []byte("sealed"), IsEnabled: true,
	}

	request := providerRequest{Kind: "facebook", Slug: "facebook", Name: new("Google")}
	if err := request.applyTo(&provider, seal, false); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	if provider.Kind != model.SocialGoogle || provider.Slug != "google" {
		t.Errorf("an update changed the kind or the slug: %s/%s", provider.Kind, provider.Slug)
	}
}

// A new secret replaces the old one, and an empty one clears it — which is
// how a provider that changes what kind of credential it uses is corrected.
func TestProviderRequestReplacesSecrets(t *testing.T) {
	seal := sealer(t)
	provider := model.SocialProvider{
		Kind: model.SocialGoogle, Slug: "google", Name: "Google", ClientID: "id",
		ClientSecret: []byte("sealed"),
	}

	request := providerRequest{ClientSecret: new("a new secret")}
	if err := request.applyTo(&provider, seal, false); err != nil {
		t.Fatalf("applyTo() = %v", err)
	}

	opened, err := seal.OpenBytes(provider.ClientSecret)
	if err != nil {
		t.Fatalf("the new secret cannot be opened: %v", err)
	}
	if string(opened) != "a new secret" {
		t.Errorf("secret = %q", opened)
	}
}
