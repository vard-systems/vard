package sso

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"loginer/internal/api/audit"
	"loginer/internal/api/respond"
	"loginer/internal/api/validate"
	"loginer/internal/jose"
	"loginer/internal/model"
	"loginer/internal/oidc"
	"loginer/internal/store"
)

type Handler struct {
	store    *store.Store
	sealer   *jose.Sealer
	provider *oidc.Service
	audit    audit.Recorder
	log      *slog.Logger
	issuer   string
}

// New returns a Handler.
func New(st *store.Store, sealer *jose.Sealer, provider *oidc.Service, recorder audit.Recorder, log *slog.Logger, issuer string) *Handler {
	return &Handler{store: st, sealer: sealer, provider: provider, audit: recorder, log: log, issuer: issuer}
}

// List returns every connection, with how many people sign in through each.
func (h *Handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	connections, err := h.store.SSOConnections(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "listing SSO connections failed")
		return
	}

	counts, err := h.store.SSOIdentityCounts(ctx)
	if err != nil {
		respond.Failure(c, h.log, err, "counting SSO identities failed")
		return
	}

	out := listResponse{Connections: make([]connectionResponse, 0, len(connections))}
	for _, connection := range connections {
		out.Connections = append(out.Connections, newConnectionResponse(connection, counts[connection.ID], h.issuer))
	}

	c.JSON(http.StatusOK, out)
}

// Get returns one connection.
func (h *Handler) Get(c *gin.Context) {
	connection, ok := h.find(c)
	if !ok {
		return
	}

	h.answer(c, http.StatusOK, connection)
}

// Create adds a connection, disabled until an administrator turns it on. A SAML
// connection gets its own signing key and certificate.
func (h *Handler) Create(c *gin.Context) {
	var req connectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	connection := &model.SSOConnection{
		Matching:     model.SSOMatchLink,
		CreateUsers:  true,
		SyncProfile:  true,
		NameIDFormat: model.SSONameIDEmail,
	}

	if err := req.applyTo(connection, h.sealer, true); err != nil {
		respond.Failure(c, h.log, err, "validating an SSO connection failed")
		return
	}

	if !h.fetchMetadata(c, connection, req.Metadata == nil) || !h.check(c, connection) {
		return
	}

	if connection.Protocol == model.SSOProtocolSAML {
		if err := h.giveKey(connection); err != nil {
			respond.Failure(c, h.log, err, "making an SSO connection's key failed")
			return
		}
	}

	if err := h.store.CreateSSOConnection(c.Request.Context(), connection); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			respond.Fail(c, slugTaken, "slug", connection.Slug)
			return
		}

		respond.Failure(c, h.log, err, "creating an SSO connection failed")
		return
	}

	h.audit.RecordWith(c, "sso_connection.created", targetType, connection.ID.String(), map[string]any{
		"connection": connection.Name, "protocol": string(connection.Protocol),
	})

	h.answer(c, http.StatusCreated, connection)
}

// Update changes a connection. Its slug and protocol are not among them: both
// are in the addresses its provider was given.
func (h *Handler) Update(c *gin.Context) {
	connection, ok := h.find(c)
	if !ok {
		return
	}

	var req connectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}

	metadataURL := connection.MetadataURL
	provider := providerOf(connection)

	if err := req.applyTo(connection, h.sealer, false); err != nil {
		respond.Failure(c, h.log, err, "validating an SSO connection failed")
		return
	}

	refetch := req.Metadata == nil && connection.MetadataURL != metadataURL
	if !h.fetchMetadata(c, connection, refetch) || !h.check(c, connection) {
		return
	}

	newProvider := providerOf(connection) != provider
	if err := h.store.SaveSSOConnection(c.Request.Context(), connection, newProvider); err != nil {
		respond.Failure(c, h.log, err, "updating an SSO connection failed")
		return
	}

	h.audit.RecordWith(c, "sso_connection.updated", targetType, connection.ID.String(), map[string]any{
		"connection": connection.Name, "is_enabled": connection.IsEnabled, "enforce_domains": connection.EnforceDomains,
		"new_provider": newProvider,
	})

	h.answer(c, http.StatusOK, connection)
}

// RefreshMetadata reads a SAML provider's metadata again from its address —
// for when it has rolled its certificate over.
func (h *Handler) RefreshMetadata(c *gin.Context) {
	connection, ok := h.find(c)
	if !ok {
		return
	}

	if connection.Protocol != model.SSOProtocolSAML || connection.MetadataURL == "" {
		respond.Fail(c, metadataInvalid, "reason", "this connection has no metadata address to read")
		return
	}

	provider := providerOf(connection)
	if !h.fetchMetadata(c, connection, true) {
		return
	}

	newProvider := providerOf(connection) != provider
	if err := h.store.SaveSSOConnection(c.Request.Context(), connection, newProvider); err != nil {
		respond.Failure(c, h.log, err, "saving refreshed SSO metadata failed")
		return
	}

	h.audit.RecordWith(c, "sso_connection.updated", targetType, connection.ID.String(), map[string]any{
		"connection": connection.Name, "metadata": "refreshed", "new_provider": newProvider,
	})

	h.answer(c, http.StatusOK, connection)
}

// Delete removes a connection and the identities held at it. The accounts
// stay, and sign in however else they can.
func (h *Handler) Delete(c *gin.Context) {
	connection, ok := h.find(c)
	if !ok {
		return
	}

	if err := h.store.DeleteSSOConnection(c.Request.Context(), connection); err != nil {
		respond.Failure(c, h.log, err, "deleting an SSO connection failed")
		return
	}

	h.audit.RecordWith(c, "sso_connection.deleted", targetType, connection.ID.String(), map[string]any{
		"connection": connection.Name,
	})

	c.Status(http.StatusNoContent)
}

// Test tries a provider before it is relied on: an OpenID Connect issuer's
// discovery, or a SAML provider's metadata, from its address or as pasted.
func (h *Handler) Test(c *gin.Context) {
	var req testRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.InvalidBody)
		return
	}
	if err := validate.Struct(req); err != nil {
		respond.Failure(c, h.log, err, "validating an SSO test failed")
		return
	}

	ctx := c.Request.Context()

	if req.Protocol == string(model.SSOProtocolOIDC) {
		document, err := h.provider.Discover(ctx, req.Issuer)
		if err != nil {
			respond.Fail(c, discoveryFailed, "reason", err.Error())
			return
		}

		c.JSON(http.StatusOK, testResponse{
			Issuer:                document.Issuer,
			AuthorizationEndpoint: document.AuthorizationEndpoint,
			TokenEndpoint:         document.TokenEndpoint,
			JWKSURI:               document.JWKSURI,
			UnsupportedScopes:     unsupportedScopes(req.Scopes, document.ScopesSupported),
		})
		return
	}

	metadata := req.Metadata
	if metadata == "" && req.MetadataURL != "" {
		fetched, err := h.provider.FetchSAMLMetadata(ctx, req.MetadataURL)
		if err != nil {
			respond.Fail(c, metadataInvalid, "reason", err.Error())
			return
		}
		metadata = fetched
	}

	descriptor, err := oidc.ParseSAMLMetadata([]byte(metadata))
	if err != nil {
		respond.Fail(c, metadataInvalid, "reason", err.Error())
		return
	}

	c.JSON(http.StatusOK, testResponse{IdentityProvider: describeIdentityProvider(descriptor), Metadata: metadata})
}

func (h *Handler) fetchMetadata(c *gin.Context, connection *model.SSOConnection, fetch bool) bool {
	if connection.Protocol != model.SSOProtocolSAML || connection.MetadataURL == "" || !fetch {
		return true
	}

	metadata, err := h.provider.FetchSAMLMetadata(c.Request.Context(), connection.MetadataURL)
	if err != nil {
		respond.Fail(c, metadataInvalid, "reason", err.Error())
		return false
	}
	connection.Metadata = metadata

	return true
}

func (h *Handler) check(c *gin.Context, connection *model.SSOConnection) bool {
	ctx := c.Request.Context()

	if connection.Protocol == model.SSOProtocolSAML {
		if _, err := oidc.ParseSAMLMetadata([]byte(connection.Metadata)); err != nil {
			respond.Fail(c, metadataInvalid, "reason", err.Error())
			return false
		}
	}

	taken, err := h.store.SSODomainsTaken(ctx, connection.Domains, connection.ID)
	if err != nil {
		respond.Failure(c, h.log, err, "checking SSO domains failed")
		return false
	}
	if len(taken) > 0 {
		respond.Fail(c, domainTaken, "domain", taken[0])
		return false
	}

	if roles := connection.RoleMappings.Roles(); len(roles) > 0 {
		found, err := h.store.UserRolesByID(ctx, roles)
		if err != nil {
			respond.Failure(c, h.log, err, "loading mapped roles failed")
			return false
		}
		for _, id := range roles {
			if !slices.ContainsFunc(found, func(role model.UserRole) bool { return role.ID == id }) {
				respond.Fail(c, roleUnknown)
				return false
			}
		}
	}

	return true
}

// giveKey makes a SAML connection's signing key and a ten-year self-signed
// certificate; it is trusted by being handed to the provider, so expiry only
// causes outages.
func (h *Handler) giveKey(connection *model.SSOConnection) error {
	key, err := jose.Generate(jose.RS256)
	if err != nil {
		return err
	}

	certificate, err := jose.SelfSigned(key, connection.EntityID(h.issuer), 10*365*24*time.Hour)
	if err != nil {
		return err
	}

	sealed, err := h.sealer.Seal(key)
	if err != nil {
		return err
	}

	connection.SPKey, connection.SPCertificate = sealed, certificate

	return nil
}

func (h *Handler) find(c *gin.Context) (*model.SSOConnection, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respond.Fail(c, connectionAbsent)
		return nil, false
	}

	connection, err := h.store.SSOConnection(c.Request.Context(), id)
	switch {
	case errors.Is(err, store.ErrNotFound):
		respond.Fail(c, connectionAbsent)
		return nil, false
	case err != nil:
		respond.Failure(c, h.log, err, "loading an SSO connection failed")
		return nil, false
	}

	return connection, true
}

func (h *Handler) answer(c *gin.Context, status int, connection *model.SSOConnection) {
	counts, err := h.store.SSOIdentityCounts(c.Request.Context())
	if err != nil {
		respond.Failure(c, h.log, err, "counting SSO identities failed")
		return
	}

	c.JSON(status, response{Connection: newConnectionResponse(*connection, counts[connection.ID], h.issuer)})
}
