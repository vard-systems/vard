package oidc

import (
	"net/url"
	"testing"
)

// A registered redirect URI's own query is the client's, and some compare
// what comes back with the address they registered: it is kept exactly as
// it is, and what this server adds comes after it, encoded.
func TestWithQueryKeepsTheAddressItsOwnQuery(t *testing.T) {
	added := url.Values{"code": {"a b+c/="}, "state": {"x&y=z"}, "iss": {"https://id.example"}}
	const tail = "code=a+b%2Bc%2F%3D&iss=https%3A%2F%2Fid.example&state=x%26y%3Dz"

	tests := []struct {
		name   string
		base   string
		values url.Values
		want   string
	}{
		{
			name:   "an address without a query",
			base:   "https://app.example/cb",
			values: added,
			want:   "https://app.example/cb?" + tail,
		},
		{
			name:   "a query out of alphabetical order",
			base:   "https://app.example/cb?b=2&a=1",
			values: added,
			want:   "https://app.example/cb?b=2&a=1&" + tail,
		},
		{
			name:   "a query spelled as the client spelled it",
			base:   "https://app.example/cb?list=a,b&path=%2Fx&flag",
			values: added,
			want:   "https://app.example/cb?list=a,b&path=%2Fx&flag&" + tail,
		},
		{
			name:   "a native app's custom scheme",
			base:   "com.example.app:/cb?b=2&a=1",
			values: url.Values{"code": {"c"}},
			want:   "com.example.app:/cb?b=2&a=1&code=c",
		},
		{
			name:   "nothing to add",
			base:   "https://app.example/cb?b=2&a=1",
			values: url.Values{"state": {""}},
			want:   "https://app.example/cb?b=2&a=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := withQuery(tt.base, tt.values)
			if got != tt.want {
				t.Fatalf("withQuery = %s\n                 want %s", got, tt.want)
			}

			parsed, err := url.Parse(got)
			if err != nil {
				t.Fatal(err)
			}
			for key, vals := range tt.values {
				if vals[0] != "" && parsed.Query().Get(key) != vals[0] {
					t.Errorf("%s reads back as %q, want %q", key, parsed.Query().Get(key), vals[0])
				}
			}
		})
	}
}
