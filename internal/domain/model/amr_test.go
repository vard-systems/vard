package model

import (
	"slices"
	"testing"
)

func TestAMRSaysHowSomeoneSignedIn(t *testing.T) {
	tests := []struct {
		name    string
		methods []string
		want    []string
	}{
		{"a password", []string{AMRPassword}, []string{AMRPassword}},
		{"a password and an emailed code", []string{AMRPassword, AMROneTimeCode}, []string{AMRPassword, AMROneTimeCode, AMRMultiFactor}},
		{"a password and a passkey as the second factor", []string{AMRPassword, AMRHardwareKey}, []string{AMRPassword, AMRHardwareKey, AMRMultiFactor}},
		{"a passkey with the person verified", []string{AMRHardwareKey, AMRUserPresence}, []string{AMRHardwareKey, AMRUserPresence, AMRMultiFactor}},
		{"a code after another provider", []string{AMROneTimeCode}, []string{AMROneTimeCode}},
		{"another provider alone", nil, nil},
		{"two codes are still one kind", []string{AMROneTimeCode, AMROneTimeCode}, []string{AMROneTimeCode}},
		{"something unknown is left out", []string{"face", AMRPassword}, []string{AMRPassword}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AMR(tt.methods); !slices.Equal(got, tt.want) {
				t.Errorf("AMR(%v) = %v, want %v", tt.methods, got, tt.want)
			}
		})
	}
}
