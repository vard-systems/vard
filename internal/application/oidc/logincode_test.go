package oidc

import "testing"

// The address the code went to is shown back so somebody can tell which of
// their addresses it was, and no more than that: an address they did not
// already know is not one they can read off the page.
func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "an ordinary address",
			email: "alexander@example.com",
			want:  "a•••••••r@example.com",
		},
		{
			name:  "a short local part keeps its ends",
			email: "abc@example.com",
			want:  "a•c@example.com",
		},
		{
			name:  "two letters give nothing away",
			email: "ab@example.com",
			want:  "••@example.com",
		},
		{
			name:  "one letter",
			email: "a@example.com",
			want:  "•@example.com",
		},
		{
			name:  "an address with an @ in the local part is masked to the last one",
			email: `"a@b"@example.com`,
			want:  `"•••"@example.com`,
		},
		{
			name:  "something that is not an address is left as it is",
			email: "not an address",
			want:  "not an address",
		},
		{
			name:  "nothing",
			email: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskEmail(tt.email); got != tt.want {
				t.Errorf("maskEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}
