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
		SecondFactor: SSOSecondFactorLocal, AcceptedAMR: DefaultAcceptedAMR, MFAContexts: StringList{},
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
			name:   "trusting an OpenID Connect provider's mfa",
			change: func(c *SSOConnection) { c.SecondFactor = SSOSecondFactorProvider },
		},
		{
			name:    "trusting a provider's second factor with no amr value to accept",
			change:  func(c *SSOConnection) { c.SecondFactor, c.AcceptedAMR = SSOSecondFactorProvider, StringList{} },
			wantErr: true,
		},
		{
			name: "trusting a SAML provider with no authentication context to accept",
			change: func(c *SSOConnection) {
				c.Protocol, c.Metadata, c.NameIDFormat = SSOProtocolSAML, "<EntityDescriptor/>", SSONameIDEmail
				c.SecondFactor = SSOSecondFactorProvider
			},
			wantErr: true,
		},
		{
			name: "trusting a SAML provider's listed context",
			change: func(c *SSOConnection) {
				c.Protocol, c.Metadata, c.NameIDFormat = SSOProtocolSAML, "<EntityDescriptor/>", SSONameIDEmail
				c.SecondFactor, c.MFAContexts = SSOSecondFactorProvider, StringList{"urn:example:mfa"}
			},
		},
		{
			name:    "a second factor nobody named",
			change:  func(c *SSOConnection) { c.SecondFactor = "both" },
			wantErr: true,
		},
		{
			name:    "an accepted value padded with spaces",
			change:  func(c *SSOConnection) { c.AcceptedAMR = StringList{" mfa"} },
			wantErr: true,
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

// A provider's answer is a second factor only in provider mode, and only by
// a value the connection lists for its protocol.
func TestAProvidersAnswerProvesASecondFactorOnlyWhereListed(t *testing.T) {
	oidc := oidcConnection()
	saml := oidcConnection()
	saml.Protocol, saml.MFAContexts = SSOProtocolSAML, StringList{"urn:example:mfa"}

	tests := []struct {
		name       string
		connection SSOConnection
		provider   bool
		said       []string
		want       bool
	}{
		{"amr mfa, provider mode", oidc, true, []string{"pwd", "mfa"}, true},
		{"amr mfa, local mode", oidc, false, []string{"mfa"}, false},
		{"amr without an accepted value", oidc, true, []string{"pwd", "otp"}, false},
		{"no amr at all", oidc, true, nil, false},
		{"a SAML context listed", saml, true, []string{"urn:example:mfa"}, true},
		{"a SAML context not listed", saml, true, []string{"urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport"}, false},
		{"amr mfa at a SAML connection, which lists contexts", saml, true, []string{"mfa"}, false},
		{"an empty value never matches", saml, true, []string{""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connection := tt.connection
			if tt.provider {
				connection.SecondFactor = SSOSecondFactorProvider
			}
			if got := connection.ProvedSecondFactor(tt.said); got != tt.want {
				t.Errorf("ProvedSecondFactor(%v) = %v, want %v", tt.said, got, tt.want)
			}
		})
	}
}

// A connection saved without its second-factor settings gets the safe ones.
func TestAConnectionSavedWithoutSecondFactorSettingsIsLocal(t *testing.T) {
	var connection SSOConnection
	if err := connection.BeforeSave(nil); err != nil {
		t.Fatal(err)
	}
	if connection.SecondFactor != SSOSecondFactorLocal || !slices.Equal(connection.AcceptedAMR, DefaultAcceptedAMR) ||
		connection.MFAContexts == nil || len(connection.MFAContexts) != 0 {
		t.Errorf("saved as %q, %v, %v; want local, [mfa], []", connection.SecondFactor, connection.AcceptedAMR, connection.MFAContexts)
	}

	trusted := SSOConnection{SecondFactor: SSOSecondFactorProvider, AcceptedAMR: StringList{"hwk"}, MFAContexts: StringList{"urn:a"}}
	if err := trusted.BeforeSave(nil); err != nil {
		t.Fatal(err)
	}
	if trusted.SecondFactor != SSOSecondFactorProvider || !slices.Equal(trusted.AcceptedAMR, StringList{"hwk"}) {
		t.Errorf("settings given were changed: %q, %v", trusted.SecondFactor, trusted.AcceptedAMR)
	}
}
