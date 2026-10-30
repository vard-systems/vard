package model

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func oidcConnection() SSOConnection {
	return SSOConnection{
		Slug: "acme", Name: "Acme", Protocol: SSOProtocolOIDC,
		Domains: StringList{"acme.com"}, Matching: SSOMatchLink,
		Issuer: "https://acme.okta.com", ClientID: "acme-console",
	}
}

func TestSSOConnectionValidate(t *testing.T) {
	role := uuid.New()

	tests := []struct {
		name    string
		change  func(*SSOConnection)
		wantErr bool
	}{
		{name: "an OpenID Connect provider", change: func(*SSOConnection) {}},
		{
			name: "a SAML provider given its metadata",
			change: func(c *SSOConnection) {
				c.Protocol, c.Metadata, c.NameIDFormat = SSOProtocolSAML, "<EntityDescriptor/>", SSONameIDEmail
			},
		},
		{
			name:   "a provider on this machine, over plain http",
			change: func(c *SSOConnection) { c.Issuer = "http://localhost:8080" },
		},
		{
			name:    "a provider elsewhere over plain http",
			change:  func(c *SSOConnection) { c.Issuer = "http://acme.okta.com" },
			wantErr: true,
		},
		{
			// With no domains it vouches for anybody, and only its button
			// leads to it.
			name: "no domain, with a button, refusing known addresses",
			change: func(c *SSOConnection) {
				c.Domains = nil
				c.ShowOnLogin = true
				c.Matching = SSOMatchDeny
			},
		},
		{
			// It would take over any account at a domain nobody else holds.
			name:    "no domain, linking to existing accounts",
			change:  func(c *SSOConnection) { c.Domains = nil; c.ShowOnLogin = true; c.Matching = SSOMatchLink },
			wantErr: true,
		},
		{
			name:    "no domain and no button, so nothing leads to it",
			change:  func(c *SSOConnection) { c.Domains = nil; c.ShowOnLogin = false },
			wantErr: true,
		},
		{
			// Requiring SSO is for the addresses at its domains; with none
			// there is nobody to require it of.
			name:    "required with no domain",
			change:  func(c *SSOConnection) { c.Domains = nil; c.ShowOnLogin = true; c.EnforceDomains = true },
			wantErr: true,
		},
		{
			name:    "a domain in capitals",
			change:  func(c *SSOConnection) { c.Domains = StringList{"Acme.com"} },
			wantErr: true,
		},
		{
			name:    "something that is not a domain",
			change:  func(c *SSOConnection) { c.Domains = StringList{"acme"} },
			wantErr: true,
		},
		{
			name:    "SAML with no metadata",
			change:  func(c *SSOConnection) { c.Protocol, c.NameIDFormat = SSOProtocolSAML, SSONameIDEmail },
			wantErr: true,
		},
		{
			name:    "a mapping without a group",
			change:  func(c *SSOConnection) { c.RoleMappings = SSORoleMappings{{RoleID: role}} },
			wantErr: true,
		},
		{
			name:    "a slug with a slash",
			change:  func(c *SSOConnection) { c.Slug = "acme/../x" },
			wantErr: true,
		},
		{
			name:    "an unknown way of matching",
			change:  func(c *SSOConnection) { c.Matching = "username" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connection := oidcConnection()
			tt.change(&connection)

			if err := connection.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() = %v, want an error: %v", err, tt.wantErr)
			}
		})
	}
}

func TestSSORoleMappingsFor(t *testing.T) {
	engineers, admins := uuid.New(), uuid.New()
	mappings := SSORoleMappings{
		{Group: "Engineering", RoleID: engineers},
		{Group: "platform", RoleID: engineers},
		{Group: "admins", RoleID: admins},
	}

	tests := []struct {
		name   string
		groups []string
		want   []uuid.UUID
	}{
		{name: "in no mapped group", groups: []string{"sales"}},
		{name: "a group in another case", groups: []string{"engineering"}, want: []uuid.UUID{engineers}},
		{name: "two groups, one role", groups: []string{"Engineering", "platform"}, want: []uuid.UUID{engineers}},
		{name: "two roles", groups: []string{"admins", "platform"}, want: []uuid.UUID{engineers, admins}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mappings.For(tt.groups)
			if len(got) != len(tt.want) {
				t.Fatalf("For(%v) = %v, want %v", tt.groups, got, tt.want)
			}
			for _, role := range tt.want {
				if !slices.Contains(got, role) {
					t.Errorf("For(%v) = %v, want %v among them", tt.groups, got, role)
				}
			}
		})
	}

	if got := mappings.Roles(); len(got) != 2 {
		t.Errorf("Roles() = %v, want each role once", got)
	}
}

func TestSSOConnectionAttribute(t *testing.T) {
	connection := oidcConnection()
	if got := connection.Attribute("email"); !slices.Equal(got, []string{"email"}) {
		t.Errorf("an OIDC connection reads the address from %v, want the standard claim", got)
	}

	connection.GroupsAttribute = " roles "
	if got := connection.Attribute("groups"); !slices.Equal(got, []string{"roles"}) {
		t.Errorf("a named attribute = %v, want only it", got)
	}

	connection.Protocol = SSOProtocolSAML
	if got := connection.Attribute("first_name"); !slices.Contains(got, "urn:oid:2.5.4.42") {
		t.Errorf("a SAML connection tries %v for a first name, want the LDAP OID among them", got)
	}
}

func TestSSOConnectionOwnsEmail(t *testing.T) {
	connection := oidcConnection()
	connection.Domains = StringList{"acme.com", "acme.co.uk"}

	for email, want := range map[string]bool{
		"ada@acme.com":        true,
		" Ada@ACME.co.uk ":    true,
		"ada@notacme.com":     false,
		"ada@sub.acme.com":    false,
		"acme.com":            false,
		"ada@acme.com@evil.x": false,
	} {
		if got := connection.OwnsEmail(email); got != want {
			t.Errorf("OwnsEmail(%q) = %v, want %v", email, got, want)
		}
	}

	connection.Domains = nil
	for email, want := range map[string]bool{
		"ada@anywhere.org": true,
		"not an address":   false,
	} {
		if got := connection.OwnsEmail(email); got != want {
			t.Errorf("with no domains, OwnsEmail(%q) = %v, want %v", email, got, want)
		}
	}
}

func TestSSOConnectionAskedScopes(t *testing.T) {
	connection := oidcConnection()
	if got := connection.AskedScopes(); !slices.Equal(got, []string{"openid", "email", "profile"}) {
		t.Errorf("default scopes = %v", got)
	}

	connection.Scopes = StringList{"email", "groups"}
	if got := connection.AskedScopes(); !slices.Equal(got, []string{"openid", "email", "groups"}) {
		t.Errorf("scopes = %v, want openid first and always", got)
	}
}
