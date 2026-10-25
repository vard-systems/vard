package organization

import (
	"errors"
	"reflect"
	"testing"

	"vard/internal/api/kit/respond"
	"vard/internal/model"
)

// stored is the organisation a request in these tests is applied to.
func stored() model.Organization {
	return model.Organization{
		Name:         "Acme",
		Slug:         "acme",
		SupportEmail: "support@acme.example",
		SupportPhone: "+996 555 123456",
		TermsURL:     "https://acme.example/terms",
		PrivacyURL:   "https://acme.example/privacy",
		Timezone:     "UTC",
	}
}

func TestOrganizationRequestApplyTo(t *testing.T) {
	tests := []struct {
		name    string
		request organizationRequest
		want    string   // the message, or "" when the request is fine
		changed []string // the settings it should report as changed
		check   func(model.Organization) error
	}{
		{
			name:    "nothing sent changes nothing",
			request: organizationRequest{},
			changed: []string{},
		},
		{
			name:    "a new name, trimmed",
			request: organizationRequest{Name: new("  Acme Inc  ")},
			changed: []string{"name"},
			check: func(o model.Organization) error {
				if o.Name != "Acme Inc" {
					return errors.New("name = " + o.Name)
				}
				return nil
			},
		},
		{
			name:    "a slug is stored lower case",
			request: organizationRequest{Slug: new("Acme-Inc")},
			changed: []string{"slug"},
			check: func(o model.Organization) error {
				if o.Slug != "acme-inc" {
					return errors.New("slug = " + o.Slug)
				}
				return nil
			},
		},
		{
			name:    "a new timezone",
			request: organizationRequest{Timezone: new("Asia/Bishkek")},
			changed: []string{"timezone"},
		},
		{
			name:    "an empty string clears a setting that may be empty",
			request: organizationRequest{SupportEmail: new(""), SupportPhone: new("")},
			changed: []string{"support_email", "support_phone"},
		},
		{
			name:    "the same values again are no change",
			request: organizationRequest{Name: new("Acme"), Slug: new("acme")},
			changed: []string{},
		},
		{
			name:    "new agreements",
			request: organizationRequest{TermsURL: new("https://acme.example/legal/terms")},
			changed: []string{"terms_url"},
		},
		{
			name:    "a name cleared",
			request: organizationRequest{Name: new("   ")},
			want:    "name is required",
		},
		{
			name:    "a slug with spaces",
			request: organizationRequest{Slug: new("acme inc")},
			want:    "slug must be lower case letters, numbers and dashes, such as acme-inc",
		},
		{
			name:    "a timezone cleared",
			request: organizationRequest{Timezone: new("")},
			want:    "timezone is required",
		},
		{
			name:    "a timezone that is an offset",
			request: organizationRequest{Timezone: new("GMT+6")},
			want:    "timezone must be a zone such as Asia/Bishkek or UTC",
		},
		{
			name:    "an address that is not one",
			request: organizationRequest{SupportEmail: new("support at acme")},
			want:    "support_email must be an email address",
		},
		{
			name:    "a support number nobody could dial",
			request: organizationRequest{SupportPhone: new("ring us")},
			want:    "support_phone must be a number someone can dial, such as +996 555 123456",
		},
		{
			name:    "terms that are not a full address",
			request: organizationRequest{TermsURL: new("example.com/terms")},
			want:    "terms_url must be a full address starting with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := tt.request
			organization := stored()

			changed, err := request.applyTo(&organization)

			if tt.want != "" {
				var fault respond.Fault
				if !errors.As(err, &fault) {
					t.Fatalf("applyTo() = %v, want a Fault", err)
				}
				if fault.Message != tt.want {
					t.Errorf("message = %q, want %q", fault.Message, tt.want)
				}
				// Nothing may be written when a request is refused.
				if !reflect.DeepEqual(organization, stored()) {
					t.Errorf("the organization was changed by a refused request")
				}
				return
			}

			if err != nil {
				t.Fatalf("applyTo() = %v, want nothing", err)
			}
			if !reflect.DeepEqual(changed, tt.changed) {
				t.Errorf("changed = %v, want %v", changed, tt.changed)
			}
			if tt.check != nil {
				if err := tt.check(organization); err != nil {
					t.Error(err)
				}
			}
		})
	}
}
