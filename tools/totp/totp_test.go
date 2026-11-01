package totp

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"

	"vard/core/brand"
)

// The SHA-1 vectors of RFC 6238 appendix B, which use eight digits; six-digit
// codes are their last six.
func TestCodeMatchesTheRFCVectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))

	vectors := map[int64]string{
		59:          "94287082",
		1111111109:  "07081804",
		1111111111:  "14050471",
		1234567890:  "89005924",
		2000000000:  "69279037",
		20000000000: "65353130",
	}

	for unix, want := range vectors {
		got, err := Code(secret, Step(time.Unix(unix, 0)))
		if err != nil {
			t.Fatal(err)
		}
		if got != want[2:] {
			t.Errorf("code at %d = %s, want %s", unix, got, want[2:])
		}
	}
}

func TestVerifyAcceptsNeighbouringStepsOnce(t *testing.T) {
	secret, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_800_000_000, 0)
	current := Step(now)

	previous, _ := Code(secret, current-1)
	next, _ := Code(secret, current+1)
	far, _ := Code(secret, current+3)

	if step, ok := Verify(secret, previous, now, 0); !ok || step != current-1 {
		t.Errorf("the previous step's code = %v, %d", ok, step)
	}
	if _, ok := Verify(secret, next[:3]+" "+next[3:], now, 0); !ok {
		t.Error("the next step's code, typed with a space, was refused")
	}
	if _, ok := Verify(secret, far, now, 0); ok {
		t.Error("a code three steps ahead was accepted")
	}

	// Once a step is used, its code — and older ones — are refused.
	if _, ok := Verify(secret, previous, now, current-1); ok {
		t.Error("a used code was accepted again")
	}
	if _, ok := Verify(secret, "12345", now, 0); ok {
		t.Error("a five-digit code was accepted")
	}
}

func TestURI(t *testing.T) {
	uri := URI(brand.Name, "root@example.com", "JBSWY3DPEHPK3PXP")

	for _, want := range []string{
		"otpauth://totp/" + brand.Name + ":root@example.com?",
		"secret=JBSWY3DPEHPK3PXP",
		"issuer=" + brand.Name,
		"digits=6",
		"period=30",
	} {
		if !strings.Contains(uri, want) {
			t.Errorf("URI %q lacks %q", uri, want)
		}
	}
}
