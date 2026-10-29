package oidc

import (
	"encoding/json"
	"testing"
)

// OpenID Connect Discovery gives some fields a default for when they are
// missing, and request_uri_parameter_supported's is true: left out, the
// document would claim request_uri values the server does not fetch. So each
// of these is written out even when false.
func TestDiscoverySaysFalseOutLoud(t *testing.T) {
	encoded, err := json.Marshal(Discovery{})
	if err != nil {
		t.Fatal(err)
	}

	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}

	for _, field := range []string{
		"request_uri_parameter_supported",
		"request_parameter_supported",
		"claims_parameter_supported",
		"require_pushed_authorization_requests",
		"authorization_response_iss_parameter_supported",
	} {
		if value, ok := document[field]; !ok || value != false {
			t.Errorf("%s = %v (present: %v), want false written out", field, value, ok)
		}
	}
}
