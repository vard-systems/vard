// Package social manages the social sign-in providers (Google, GitHub, Apple…).
package social

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"vard/internal/api/kit/audit"
	"vard/internal/api/kit/respond"
	"vard/internal/jose"
	"vard/internal/model"
	"vard/internal/store"
)

type Handler struct {
	store  *store.Store
	sealer *jose.Sealer
	audit  audit.Recorder
	log    *slog.Logger
	issuer string
}

func New(st *store.Store, sealer *jose.Sealer, recorder audit.Recorder, log *slog.Logger, issuer string) *Handler {
	return &Handler{store: st, sealer: sealer, audit: recorder, log: log, issuer: issuer}
}

// List returns every configured provider, and the kinds a new one may be.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	providers, err := h.store.SocialProviders(ctx, false)
	if err != nil {
		respond.Failure(c, h.log, err, "listing social providers failed")
		return
	}

	counts, err := h.store.SocialIdentityCounts(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "counting social identities failed")
		return
	}

	c.JSON(http.StatusOK, newListResponse(providers, counts, h.issuer))
}

func (h *Handler) Get(c *gin.Context) {
	provider, ok := h.find(c)
	if !ok {
		return
	}

	h.answer(c, http.StatusOK, provider)
}

// Create registers a provider. Its credentials are proved only by a first
// sign-in.
func (h *Handler) Create(c *gin.Context) {
	var req providerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	provider := &model.SocialProvider{
		IsEnabled:          true,
		LinkVerifiedEmails: true,
		AllowRegistration:  true,
	}

	if err := req.applyTo(provider, h.sealer, true); err != nil {
		respond.Failure(c, h.log, err, "validating a social provider failed")
		return
	}

	if req.Position == nil {
		provider.Position = h.store.NextSocialPosition(c.Request.Context())
	}

	if err := h.store.CreateSocialProvider(c.Request.Context(), provider); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			respond.Fail(c, slugTaken)
			return
		}

		respond.Failure(c, h.log, err, "creating social provider failed")
		return
	}

	h.audit.RecordWith(c, "social_provider.created", targetType, provider.ID.String(), map[string]any{
		"provider": provider.Name, "kind": string(provider.Kind),
	})

	h.answer(c, http.StatusCreated, provider)
}

// Update changes a provider's settings. Kind and slug are fixed, since they
// form the registered callback address.
func (h *Handler) Update(c *gin.Context) {
	provider, ok := h.find(c)
	if !ok {
		return
	}

	var req providerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	before := identityNamespace(provider)

	if err := req.applyTo(provider, h.sealer, false); err != nil {
		respond.Failure(c, h.log, err, "validating a social provider failed")
		return
	}

	// Pointing a provider at a different upstream changes whose subjects its
	// identities name, so a stranger there could sign in as an existing user.
	// The change is refused while identities remain; register a new provider
	// instead.
	if identityNamespace(provider) != before {
		counts, err := h.store.SocialIdentityCounts(c.Request.Context())
		if err != nil {
			respond.Failure(c, h.log, err, "counting social identities failed")
			return
		}
		if counts[provider.ID] > 0 {
			respond.Fail(c, providerRepointed)
			return
		}
	}

	if err := h.store.SaveSocialProvider(c.Request.Context(), provider); err != nil {
		respond.Failure(c, h.log, err, "updating social provider failed")
		return
	}

	h.audit.RecordWith(c, "social_provider.updated", targetType, provider.ID.String(), map[string]any{
		"provider": provider.Name, "is_enabled": provider.IsEnabled,
	})

	h.answer(c, http.StatusOK, provider)
}

var providerRepointed = respond.Define(http.StatusConflict, "social_provider_repointed", respond.Admin)

func identityNamespace(p *model.SocialProvider) string {
	_, token, userInfo := p.Endpoints()

	switch p.Kind {
	case model.SocialOAuth2, model.SocialOIDC:
		return "custom\x00" + token + "\x00" + userInfo + "\x00" + p.ClientID
	case model.SocialApple:
		return "apple\x00" + p.TeamID
	case model.SocialFacebook:
		return "facebook\x00" + p.ClientID
	default:
		return ""
	}
}

// Secret returns the stored client secret so it can be checked against the
// provider's console. It requires the permission that could replace it, and
// every read is written to the activity log.
func (h *Handler) Secret(c *gin.Context) {
	provider, ok := h.find(c)
	if !ok {
		return
	}

	sealed, what, where := provider.ClientSecret, "client_secret", provider.ClientSecretWhere()
	if provider.Spec().SignedSecret {
		sealed, what, where = provider.PrivateKey, "private_key", provider.PrivateKeyWhere()
	}

	if len(sealed) == 0 {
		respond.Fail(c, noSecret)
		return
	}

	secret, _, err := h.sealer.OpenFor(where, sealed)
	if err != nil {
		respond.Failure(c, h.log, err, "opening a social provider secret failed")
		return
	}

	h.audit.RecordWith(c, "social_provider.secret_read", targetType, provider.ID.String(), map[string]any{
		"provider": provider.Name, "secret": what,
	})

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"secret": string(secret), "kind": what})
}

// Delete removes a provider and its identities; the accounts stay, though some
// may lose their only way in.
func (h *Handler) Delete(c *gin.Context) {
	provider, ok := h.find(c)
	if !ok {
		return
	}

	if err := h.store.DeleteSocialProvider(c.Request.Context(), provider); err != nil {
		respond.Failure(c, h.log, err, "deleting social provider failed")
		return
	}

	h.audit.RecordWith(c, "social_provider.deleted", targetType, provider.ID.String(), map[string]any{
		"provider": provider.Name,
	})

	c.Status(http.StatusNoContent)
}

func (h *Handler) find(c *gin.Context) (*model.SocialProvider, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, providerAbsent)
		return nil, false
	}

	provider, err := h.store.SocialProvider(c.Request.Context(), id)

	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, providerAbsent)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading social provider failed")
		return nil, false
	}

	return provider, true
}

func (h *Handler) answer(c *gin.Context, status int, provider *model.SocialProvider) {
	counts, err := h.store.SocialIdentityCounts(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "counting social identities failed")
		return
	}

	c.JSON(status, gin.H{"provider": newProviderResponse(*provider, h.issuer, counts[provider.ID])})
}
