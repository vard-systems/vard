package model

import "testing"

func TestOrganizationValidate(t *testing.T) {
	valid := func() Organization {
		return Organization{
			Name:         "Acme",
			Slug:         "acme-inc",
			SupportEmail: "support@acme.example.com",
			SupportPhone: "+996 555 123456",
			TermsURL:     "https://acme.example.com/terms",
			PrivacyURL:   "https://acme.example.com/privacy",
			Timezone:     "Asia/Bishkek",
		}
	}

	tests := []struct {
		name   string
		change func(*Organization)
		want   string // the message, or "" when it is fine
	}{
		{name: "everything filled in", change: func(*Organization) {}},
		{
			name: "nothing optional filled in",
			change: func(o *Organization) {
				o.SupportEmail, o.SupportPhone = "", ""
				o.TermsURL, o.PrivacyURL = "", ""
			},
		},
		{name: "no name", change: func(o *Organization) { o.Name = " " }, want: "name is required"},
		{
			name:   "no slug",
			change: func(o *Organization) { o.Slug = "" },
			want:   "slug is required",
		},
		{
			name:   "a slug that ends in a dash",
			change: func(o *Organization) { o.Slug = "acme-" },
			want:   "slug must be lower case letters, numbers and dashes, such as acme-inc",
		},
		{
			name:   "no timezone",
			change: func(o *Organization) { o.Timezone = "" },
			want:   "timezone is required",
		},
		{
			name:   "a timezone the database does not name",
			change: func(o *Organization) { o.Timezone = "Mars/Olympus" },
			want:   "timezone must be a zone such as Asia/Bishkek or UTC",
		},
		{
			name:   "the server's own zone, which nobody chose",
			change: func(o *Organization) { o.Timezone = "Local" },
			want:   "timezone must be a zone such as Asia/Bishkek or UTC",
		},
		{
			name:   "a number written for people to read",
			change: func(o *Organization) { o.SupportPhone = "+1 (555) 010-9999" },
		},
		{
			name:   "a support number with letters in it",
			change: func(o *Organization) { o.SupportPhone = "call us" },
			want:   "support_phone must be a number someone can dial, such as +996 555 123456",
		},
		{
			name:   "terms that are not a full address",
			change: func(o *Organization) { o.TermsURL = "/terms" },
			want:   "terms_url must be a full address starting with http:// or https://",
		},
		{
			name:   "a privacy link that could run something",
			change: func(o *Organization) { o.PrivacyURL = "javascript:alert(1)" },
			want:   "privacy_url must be a full address starting with http:// or https://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			organization := valid()
			tt.change(&organization)

			err := organization.Validate()

			switch {
			case tt.want == "" && err != nil:
				t.Fatalf("Validate() = %v, want nothing", err)
			case tt.want != "" && err == nil:
				t.Fatalf("Validate() = nothing, want %q", tt.want)
			case tt.want != "" && err.Error() != tt.want:
				t.Errorf("Validate() = %q, want %q", err, tt.want)
			}
		})
	}
}

// The row a fresh installation starts with has to pass the rules the panel
// holds every later change to, or the settings page opens on something it
// would refuse to save back.
func TestDefaultOrganizationIsValid(t *testing.T) {
	if err := DefaultOrganization().Validate(); err != nil {
		t.Fatalf("DefaultOrganization() is not valid: %v", err)
	}
}
