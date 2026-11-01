package sso

import (
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/google/uuid"

	"vard/api/kit/respond"
	"vard/api/kit/validate"
	"vard/core/model"
	"vard/core/oidc"
	"vard/tools/jose"
)

func (r *connectionRequest) applyTo(connection *model.SSOConnection, sealer *jose.Sealer, creating bool) error {
	if err := validate.Struct(r); err != nil {
		return err
	}

	if creating {
		// Its id before it is stored, so its secrets are sealed to it.
		if connection.ID == uuid.Nil {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			connection.ID = id
		}

		connection.Protocol = model.SSOProtocol(validate.Text(r.Protocol, ""))
		connection.Slug = slugFrom(validate.Text(r.Slug, ""), validate.Text(r.Name, ""))

		if connection.Protocol == "" {
			return required("protocol")
		}
		if !slugPattern.MatchString(connection.Slug) {
			return invalid("slug")
		}
	}

	connection.Name = validate.Text(r.Name, connection.Name)
	connection.IsEnabled = validate.Flag(r.IsEnabled, connection.IsEnabled)
	connection.EnforceDomains = validate.Flag(r.EnforceDomains, connection.EnforceDomains)
	connection.ShowOnLogin = validate.Flag(r.ShowOnLogin, connection.ShowOnLogin)

	if r.Domains != nil {
		domains := model.StringList{}
		for _, domain := range *r.Domains {
			domain = model.NormalizeDomain(domain)
			if domain == "" || slices.Contains(domains, domain) {
				continue
			}
			if !model.ValidDomain(domain) {
				return domainInvalid.With("domain", domain)
			}
			domains = append(domains, domain)
		}
		connection.Domains = domains
	}

	connection.Issuer = strings.TrimRight(validate.Text(r.Issuer, connection.Issuer), "/")
	connection.ClientID = validate.Text(r.ClientID, connection.ClientID)

	if r.ClientSecret != nil && strings.TrimSpace(*r.ClientSecret) != "" {
		sealed, err := sealer.SealFor(connection.ClientSecretWhere(), []byte(strings.TrimSpace(*r.ClientSecret)))
		if err != nil {
			return err
		}
		connection.ClientSecret = sealed
	}

	if r.Scopes != nil {
		scopes := model.StringList{}
		for _, scope := range *r.Scopes {
			for piece := range strings.FieldsSeq(scope) {
				if !slices.Contains(scopes, piece) {
					scopes = append(scopes, piece)
				}
			}
		}
		connection.Scopes = scopes
	}

	connection.MetadataURL = validate.Text(r.MetadataURL, connection.MetadataURL)
	if r.Metadata != nil {
		connection.Metadata = strings.TrimSpace(*r.Metadata)
	}
	connection.NameIDFormat = model.SSONameIDFormat(validate.Text(r.NameIDFormat, string(connection.NameIDFormat)))
	connection.SignRequests = validate.Flag(r.SignRequests, connection.SignRequests)

	connection.Matching = model.SSOMatching(validate.Text(r.Matching, string(connection.Matching)))
	connection.CreateUsers = validate.Flag(r.CreateUsers, connection.CreateUsers)
	connection.SyncProfile = validate.Flag(r.SyncProfile, connection.SyncProfile)
	connection.SyncRoles = validate.Flag(r.SyncRoles, connection.SyncRoles)

	connection.EmailAttribute = validate.Text(r.EmailAttribute, connection.EmailAttribute)
	connection.FirstNameAttribute = validate.Text(r.FirstNameAttribute, connection.FirstNameAttribute)
	connection.LastNameAttribute = validate.Text(r.LastNameAttribute, connection.LastNameAttribute)
	connection.GroupsAttribute = validate.Text(r.GroupsAttribute, connection.GroupsAttribute)

	connection.SecondFactor = model.SSOSecondFactor(validate.Text(r.SecondFactor, string(connection.SecondFactor)))
	if r.AcceptedAMR != nil {
		connection.AcceptedAMR = acceptedValues(*r.AcceptedAMR)
	}
	if r.MFAContexts != nil {
		connection.MFAContexts = acceptedValues(*r.MFAContexts)
	}

	if r.RoleMappings != nil {
		mappings := model.SSORoleMappings{}
		for _, mapping := range *r.RoleMappings {
			id, err := uuid.Parse(mapping.RoleID)
			group := strings.TrimSpace(mapping.Group)
			if err != nil || group == "" || len(group) > 255 {
				return mappingInvalid.With("group", group)
			}
			mappings = append(mappings, model.SSORoleMapping{Group: group, RoleID: id})
		}
		connection.RoleMappings = mappings
	}

	return settle(connection)
}

func settle(connection *model.SSOConnection) error {
	// Without domains no address leads to the connection, so it needs its
	// button — and there is nobody to require it of.
	if len(connection.Domains) == 0 && (connection.EnforceDomains || !connection.ShowOnLogin) {
		return unreachable.Fault(nil)
	}

	switch connection.Protocol {
	case model.SSOProtocolOIDC:
		parsed, err := url.Parse(connection.Issuer)
		switch {
		case connection.Issuer == "":
			return required("issuer")
		case err != nil || parsed.Host == "" || parsed.Scheme != "https" && !model.IsLocalhost(parsed):
			return issuerInvalid.Fault(nil)
		case connection.ClientID == "":
			return required("client_id")
		}
	case model.SSOProtocolSAML:
		if connection.Metadata == "" && connection.MetadataURL == "" {
			return required("metadata")
		}
		if connection.NameIDFormat == "" {
			connection.NameIDFormat = model.SSONameIDEmail
		}
	}

	// Linking takes accounts over, so only a connection that names the
	// domains it answers for does it, and without a word that is the only
	// one that links.
	switch {
	case connection.Matching == "" && len(connection.Domains) == 0:
		connection.Matching = model.SSOMatchDeny
	case connection.Matching == "":
		connection.Matching = model.SSOMatchLink
	case connection.Matching == model.SSOMatchLink && len(connection.Domains) == 0:
		return linkNeedsDomains.Fault(nil)
	}
	if connection.Domains == nil {
		connection.Domains = model.StringList{}
	}
	if connection.RoleMappings == nil {
		connection.RoleMappings = model.SSORoleMappings{}
	}
	if connection.SecondFactor == "" {
		connection.SecondFactor = model.SSOSecondFactorLocal
	}
	if connection.AcceptedAMR == nil {
		connection.AcceptedAMR = slices.Clone(model.DefaultAcceptedAMR)
	}
	if connection.MFAContexts == nil {
		connection.MFAContexts = model.StringList{}
	}

	if err := connection.Validate(); err != nil {
		return respond.Invalid(err.Error())
	}

	return nil
}

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$`)
	notSlug     = regexp.MustCompile(`[^a-z0-9]+`)

	requiredField = func() respond.Problem { p, _ := respond.Lookup("validation.required"); return p }()
	invalidField  = func() respond.Problem { p, _ := respond.Lookup("validation.invalid"); return p }()
)

func slugFrom(sent, name string) string {
	slug := strings.ToLower(strings.TrimSpace(sent))
	if slug == "" {
		slug = strings.Trim(notSlug.ReplaceAllString(strings.ToLower(name), "-"), "-")
	}

	if len(slug) > 64 {
		slug = strings.TrimRight(slug[:64], "-")
	}

	return slug
}

// acceptedValues are the values typed, trimmed, each once; blank ones are
// dropped.
func acceptedValues(typed []string) model.StringList {
	values := model.StringList{}
	for _, value := range typed {
		value = strings.TrimSpace(value)
		if value != "" && !slices.Contains(values, value) {
			values = append(values, value)
		}
	}
	return values
}

func required(field string) respond.Fault { return requiredField.With("field", field) }
func invalid(field string) respond.Fault  { return invalidField.With("field", field) }

func unsupportedScopes(scopes, supported []string) []string {
	if len(supported) == 0 {
		return nil
	}

	var out []string
	for _, scope := range (model.SSOConnection{Scopes: scopes}).AskedScopes() {
		if !slices.Contains(supported, scope) {
			out = append(out, scope)
		}
	}

	return out
}

// providerOf is who a connection trusts (OIDC issuer or SAML entity ID).
// Subjects are unique only within it, so changing it resets identities.
func providerOf(connection *model.SSOConnection) string {
	if connection.Protocol == model.SSOProtocolOIDC {
		return connection.Issuer
	}

	descriptor, err := oidc.ParseSAMLMetadata([]byte(connection.Metadata))
	if err != nil {
		return ""
	}

	return descriptor.EntityID
}
