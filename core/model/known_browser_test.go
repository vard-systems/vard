package model

import (
	"strings"
	"testing"
	"time"

	"vard/tools/jose"
)

// A known-browser cookie names the browser only for the account and purpose
// it was issued for, and only while it lasts; anything changed in it names
// nobody.
func TestAKnownBrowserIsKnownOnlyToItsAccount(t *testing.T) {
	sealer, err := jose.NewSealer("a-test-secret-key-0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()

	value, err := NewKnownBrowser(sealer, "users", "ada@example.com", now)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Split(value, ".")[0]

	other, err := jose.NewSealer("another-installations-key-0123456789")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(value, ".")

	cases := []struct {
		name    string
		sealer  *jose.Sealer
		purpose string
		value   string
		account string
		at      time.Time
		want    string
	}{
		{"its own account", sealer, "users", value, "ada@example.com", now, id},
		{"near the end of its life", sealer, "users", value, "ada@example.com", now.Add(KnownBrowserLifetime - time.Minute), id},
		{"another account", sealer, "users", value, "grace@example.com", now, ""},
		{"another purpose", sealer, "admins", value, "ada@example.com", now, ""},
		{"another installation", other, "users", value, "ada@example.com", now, ""},
		{"past its life", sealer, "users", value, "ada@example.com", now.Add(KnownBrowserLifetime + time.Minute), ""},
		{"issued in the future", sealer, "users", value, "ada@example.com", now.Add(-time.Hour), ""},
		{"another id with its tag", sealer, "users", "AAAAAAAAAAAAAAAAAAAAAA." + parts[1] + "." + parts[2], "ada@example.com", now, ""},
		{"a later issue time with its tag", sealer, "users", parts[0] + ".9999999999." + parts[2], "ada@example.com", now, ""},
		{"no tag", sealer, "users", parts[0] + "." + parts[1] + ".", "ada@example.com", now, ""},
		{"nothing", sealer, "users", "", "ada@example.com", now, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := KnownBrowser(c.sealer, c.purpose, c.value, c.account, c.at); got != c.want {
				t.Errorf("KnownBrowser() = %q, want %q", got, c.want)
			}
		})
	}
}
