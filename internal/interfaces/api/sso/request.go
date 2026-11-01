package sso

import (
	"net/http"

	"vard/internal/interfaces/api/kit/respond"
)

// targetType is what a connection is called in the activity log.
const targetType = "sso_connection"

// What these endpoints refuse, as the SSO page shows it.
var (
	slugTaken        = respond.Define(http.StatusConflict, "sso_slug_taken", respond.Admin)
	domainTaken      = respond.Define(http.StatusConflict, "sso_domain_taken", respond.Admin)
	domainInvalid    = respond.Define(http.StatusBadRequest, "sso_domain_invalid", respond.Admin)
	issuerInvalid    = respond.Define(http.StatusBadRequest, "sso_issuer_invalid", respond.Admin)
	discoveryFailed  = respond.Define(http.StatusBadRequest, "sso_discovery_failed", respond.Admin)
	metadataInvalid  = respond.Define(http.StatusBadRequest, "sso_metadata_invalid", respond.Admin)
	roleUnknown      = respond.Define(http.StatusBadRequest, "sso_role_unknown", respond.Admin)
	mappingInvalid   = respond.Define(http.StatusBadRequest, "sso_role_mapping_invalid", respond.Admin)
	connectionAbsent = respond.Define(http.StatusNotFound, "sso_connection_not_found", respond.Admin)
	unreachable      = respond.Define(http.StatusBadRequest, "sso_unreachable", respond.Admin)
	linkNeedsDomains = respond.Define(http.StatusBadRequest, "sso_link_needs_domains", respond.Admin)
	domainUnknown    = respond.Define(http.StatusNotFound, "sso_domain_not_listed", respond.Admin)
)

// connectionRequest is a PATCH: nil leaves a setting alone. Slug and protocol
// are read only on create, and an update without a client secret keeps the
// stored one.
type connectionRequest struct {
	Name      *string `json:"name" validate:"omitnil,min=1,max=100"`
	Slug      *string `json:"slug" validate:"omitnil,max=64"`
	Protocol  *string `json:"protocol" validate:"omitnil,oneof=oidc saml"`
	IsEnabled *bool   `json:"is_enabled"`

	Domains        *[]string `json:"domains" validate:"omitnil,max=50"`
	EnforceDomains *bool     `json:"enforce_domains"`
	ShowOnLogin    *bool     `json:"show_on_login"`

	Issuer       *string   `json:"issuer" validate:"omitnil,max=512"`
	ClientID     *string   `json:"client_id" validate:"omitnil,max=255"`
	ClientSecret *string   `json:"client_secret" validate:"omitnil,max=1024"`
	Scopes       *[]string `json:"scopes" validate:"omitnil,max=20"`

	MetadataURL  *string `json:"metadata_url" validate:"omitnil,max=1024"`
	Metadata     *string `json:"metadata" validate:"omitnil,max=200000"`
	NameIDFormat *string `json:"name_id_format" validate:"omitnil,oneof=email persistent unspecified"`
	SignRequests *bool   `json:"sign_requests"`

	Matching    *string `json:"matching" validate:"omitnil,oneof=link deny"`
	CreateUsers *bool   `json:"create_users"`
	SyncProfile *bool   `json:"sync_profile"`
	SyncRoles   *bool   `json:"sync_roles"`

	EmailAttribute     *string `json:"email_attribute" validate:"omitnil,max=255"`
	FirstNameAttribute *string `json:"first_name_attribute" validate:"omitnil,max=255"`
	LastNameAttribute  *string `json:"last_name_attribute" validate:"omitnil,max=255"`
	GroupsAttribute    *string `json:"groups_attribute" validate:"omitnil,max=255"`

	RoleMappings *[]roleMapping `json:"role_mappings" validate:"omitnil,max=100"`

	SecondFactor *string   `json:"second_factor" validate:"omitnil,oneof=local provider"`
	AcceptedAMR  *[]string `json:"accepted_amr" validate:"omitnil,max=20,dive,max=255"`
	MFAContexts  *[]string `json:"mfa_contexts" validate:"omitnil,max=20,dive,max=255"`
}

type roleMapping struct {
	Group  string `json:"group"`
	RoleID string `json:"role_id"`
}

// testRequest is a connection's provider to try before saving: an issuer to
// discover, or metadata to read.
type testRequest struct {
	Protocol    string   `json:"protocol" validate:"required,oneof=oidc saml"`
	Issuer      string   `json:"issuer" validate:"max=512"`
	Scopes      []string `json:"scopes" validate:"max=32,dive,max=128"`
	MetadataURL string   `json:"metadata_url" validate:"max=1024"`
	Metadata    string   `json:"metadata" validate:"max=200000"`
}
