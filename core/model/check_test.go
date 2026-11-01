package model

import (
	"fmt"
	"strings"
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

// Each enum-like column's CHECK lists exactly the values Go accepts, so the
// database refuses what Go would and nothing Go writes; a value added in Go
// without its migration fails here.
func TestEachCheckListsWhatGoAccepts(t *testing.T) {
	tests := []struct {
		model  any
		field  string
		column string
		want   []string
	}{
		{&Admin{}, "Status", "status", texts(Statuses)},
		{&Application{}, "Type", "type", texts(ApplicationTypes)},
		{&Application{}, "TokenAuthMethod", "token_auth_method", texts(AuthMethods)},
		{&DeviceAuthorization{}, "Status", "status", texts([]DeviceStatus{DevicePending, DeviceApproved, DeviceDenied})},
		{&MailSettings{}, "Encryption", "encryption", texts(MailEncryptions)},
		{&MFA{}, "Method", "method", texts(MFAMethods)},
		{&UserFactor{}, "Method", "method", texts(MFAMethods)},
		{&SSOConnection{}, "Protocol", "protocol", texts([]SSOProtocol{SSOProtocolOIDC, SSOProtocolSAML})},
		{&SSOConnection{}, "NameIDFormat", "name_id_format", texts([]SSONameIDFormat{"", SSONameIDEmail, SSONameIDPersistent, SSONameIDUnspecified})},
		{&SSOConnection{}, "Matching", "matching", texts([]SSOMatching{SSOMatchLink, SSOMatchDeny})},
		{&SSOConnection{}, "SecondFactor", "second_factor", texts([]SSOSecondFactor{SSOSecondFactorLocal, SSOSecondFactorProvider})},
		{&SocialProvider{}, "Kind", "kind", texts([]SocialKind{SocialGoogle, SocialApple, SocialFacebook, SocialYandex, SocialVK, SocialOIDC})},
		{&SocialProvider{}, "TokenAuth", "token_auth", append([]string{""}, texts(SocialTokenAuths)...)},
		{&UserField{}, "Type", "type", texts(FieldTypes)},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T.%s", tt.model, tt.field), func(t *testing.T) {
			parsed, err := schema.Parse(tt.model, &sync.Map{}, schema.NamingStrategy{})
			if err != nil {
				t.Fatal(err)
			}
			check := parsed.LookUpField(tt.field).TagSettings["CHECK"]

			quoted := make([]string, len(tt.want))
			for i, v := range tt.want {
				quoted[i] = "'" + v + "'"
			}
			want := tt.column + " IN (" + strings.Join(quoted, ",") + ")"
			if _, got, _ := strings.Cut(check, ","); got != want {
				t.Errorf("check = %q, want %q", got, want)
			}
		})
	}
}

func texts[T ~string](list []T) []string {
	out := make([]string, len(list))
	for i, v := range list {
		out[i] = string(v)
	}
	return out
}
