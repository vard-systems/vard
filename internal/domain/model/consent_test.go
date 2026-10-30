package model

import (
	"testing"
	"time"
)

func TestConsentCovers(t *testing.T) {
	given := Consent{Scopes: []string{ScopeOpenID, ScopeEmail}}

	tests := []struct {
		name   string
		scopes []string
		want   bool
	}{
		{"what was allowed", []string{ScopeOpenID, ScopeEmail}, true},
		{"less than was allowed", []string{ScopeOpenID}, true},
		{"nothing at all", nil, true},
		{"one scope more", []string{ScopeOpenID, ScopeEmail, ScopeOfflineAccess}, false},
		{"an API scope never allowed", []string{"orders:write"}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := given.Covers(test.scopes); got != test.want {
				t.Errorf("Covers(%v) = %v, want %v", test.scopes, got, test.want)
			}
		})
	}
}

func TestConsentAllowKeepsWhatWasAllowed(t *testing.T) {
	consent := Consent{Scopes: []string{ScopeOpenID, "orders:read"}}
	consent.Allow([]string{ScopeOpenID, ScopeEmail})

	if !consent.Covers([]string{ScopeOpenID, ScopeEmail, "orders:read"}) || len(consent.Scopes) != 3 {
		t.Errorf("scopes = %v, want the old and the new once each", consent.Scopes)
	}
}

func TestOrganizationAgreementsAccepted(t *testing.T) {
	changed := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	before, after := changed.Add(-time.Hour), changed.Add(time.Hour)

	tests := []struct {
		name         string
		organization Organization
		accepted     *time.Time
		want         bool
	}{
		{"no links: nothing to accept", Organization{}, nil, true},
		{"links never accepted", Organization{TermsURL: "https://example.com/terms"}, nil, false},
		{"links accepted, never changed", Organization{PrivacyURL: "https://example.com/privacy"}, &before, true},
		{"accepted after the last change", Organization{TermsURL: "https://example.com/terms", AgreementsUpdatedAt: &changed}, &after, true},
		{"accepted before the last change", Organization{TermsURL: "https://example.com/terms", AgreementsUpdatedAt: &changed}, &before, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.organization.AgreementsAccepted(test.accepted); got != test.want {
				t.Errorf("AgreementsAccepted = %v, want %v", got, test.want)
			}
		})
	}
}
