package social

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"vard/api/kit/respond"
	"vard/api/kit/validate"
	"vard/core/model"
	"vard/tools/jose"
)

var (
	slugTaken      = respond.Define(http.StatusConflict, "social_provider_slug_taken", respond.Admin)
	noSecret       = respond.Define(http.StatusNotFound, "social_provider_secret_missing", respond.Admin)
	providerAbsent = respond.Define(http.StatusNotFound, "social_provider_not_found", respond.Admin)
	kindUnknown    = respond.Define(http.StatusBadRequest, "social_provider_kind_invalid", respond.Admin)
)

func (r *providerRequest) applyTo(provider *model.SocialProvider, sealer *jose.Sealer, creating bool) error {
	if creating {
		kind := model.SocialKind(strings.ToLower(strings.TrimSpace(r.Kind)))
		if _, known := model.SocialSpecFor(kind); !known {
			return kindUnknown.With("kinds", strings.Join(kindNames(), ", "))
		}

		provider.Kind = kind
		provider.Slug = lower(&r.Slug, string(kind))

		// Its id before it is stored, so its secrets are sealed to it.
		if provider.ID == uuid.Nil {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			provider.ID = id
		}
	}

	provider.Name = validate.Text(r.Name, provider.Name)
	provider.ClientID = validate.Text(r.ClientID, provider.ClientID)
	provider.TeamID = validate.Text(r.TeamID, provider.TeamID)
	provider.KeyID = validate.Text(r.KeyID, provider.KeyID)
	provider.AuthorizeURL = validate.Text(r.AuthorizeURL, provider.AuthorizeURL)
	provider.TokenURL = validate.Text(r.TokenURL, provider.TokenURL)
	provider.UserinfoURL = validate.Text(r.UserInfoURL, provider.UserinfoURL)
	provider.TokenAuth = model.SocialTokenAuth(lower(r.TokenAuth, string(provider.TokenAuth)))

	provider.IsEnabled = validate.Flag(r.IsEnabled, provider.IsEnabled)
	provider.LinkVerifiedEmails = validate.Flag(r.LinkVerifiedEmails, provider.LinkVerifiedEmails)
	provider.AllowRegistration = validate.Flag(r.AllowRegistration, provider.AllowRegistration)

	if r.Scopes != nil {
		provider.Scopes = cleanScopes(*r.Scopes)
	}

	if r.Position != nil {
		provider.Position = *r.Position
	}

	// A sent secret replaces the stored one; an absent one keeps it; "" clears
	// it.
	if err := seal(sealer, provider.ClientSecretWhere(), r.ClientSecret, &provider.ClientSecret); err != nil {
		return err
	}
	if err := seal(sealer, provider.PrivateKeyWhere(), r.PrivateKey, &provider.PrivateKey); err != nil {
		return err
	}

	if err := provider.Validate(); err != nil {
		return respond.Invalid(err.Error())
	}

	return nil
}

func lower(sent *string, current string) string {
	if value := strings.ToLower(validate.Text(sent, current)); value != "" {
		return value
	}

	return strings.ToLower(current)
}

func cleanScopes(sent []string) model.StringList {
	scopes := make(model.StringList, 0, len(sent))
	for _, scope := range sent {
		if scope = strings.TrimSpace(scope); scope != "" {
			scopes = append(scopes, scope)
		}
	}

	return scopes
}

func seal(sealer *jose.Sealer, where jose.Context, sent *string, stored *[]byte) error {
	if sent == nil {
		return nil
	}

	value := strings.TrimSpace(*sent)
	if value == "" {
		*stored = nil
		return nil
	}

	sealed, err := sealer.SealFor(where, []byte(value))
	if err != nil {
		return err
	}

	*stored = sealed

	return nil
}

func kindNames() []string {
	names := make([]string, 0, len(model.SocialSpecs))
	for _, spec := range model.SocialSpecs {
		names = append(names, string(spec.Kind))
	}

	return names
}
