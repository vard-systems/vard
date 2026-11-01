package model

import (
	"strings"
	"testing"
)

func TestUserCodesAreReadableAndTypable(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		code, err := NewUserCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 9 || code[4] != '-' {
			t.Fatalf("code %q is not two groups of four", code)
		}
		for _, letter := range strings.ReplaceAll(code, "-", "") {
			if !strings.ContainsRune(userCodeAlphabet, letter) {
				t.Fatalf("code %q has %q, which is not in the alphabet", code, letter)
			}
		}
		seen[code] = true
	}
	if len(seen) < 195 {
		t.Errorf("200 codes had only %d different ones", len(seen))
	}

	for _, typed := range []string{"bcdf-ghjk", " BCDF GHJK ", "BCDFGHJK"} {
		if got := NormalizeUserCode(typed); got != "BCDFGHJK" {
			t.Errorf("NormalizeUserCode(%q) = %q, want BCDFGHJK", typed, got)
		}
	}
}
