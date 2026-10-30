package model

import "testing"

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name, email, want string
	}{
		{name: "capitals", email: "Ada.Lovelace@Example.COM", want: "ada.lovelace@example.com"},
		{name: "spaces around it", email: "  ada@example.com\t", want: "ada@example.com"},
		{name: "already normal", email: "ada@example.com", want: "ada@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeEmail(tt.email); got != tt.want {
				t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}
