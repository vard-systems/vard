package oidc

import (
	"context"
	"testing"
	"time"

	"vard/internal/infrastructure/database/databasetest"
	"vard/internal/infrastructure/store"
	"vard/internal/jose"
)

// testStore is a store on a throwaway database of its own.
func testStore(t *testing.T) *store.Store {
	t.Helper()

	db, _ := databasetest.Open(t)

	return store.New(db)
}

// clock is a time the test moves by hand.
type clock struct{ at time.Time }

func (c *clock) now() time.Time          { return c.at }
func (c *clock) advance(d time.Duration) { c.at = c.at.Add(d) }

func states(ks *keySet, alg string) map[string]string {
	out := map[string]string{}
	for _, key := range ks.describe() {
		if key.Algorithm == alg {
			out[key.KID] = key.State
		}
	}
	return out
}

func count(m map[string]string, state string) int {
	n := 0
	for _, s := range m {
		if s == state {
			n++
		}
	}
	return n
}

func TestLiveKeysRotateWithALeadAndARetention(t *testing.T) {
	st := testStore(t)
	sealer, _ := jose.NewSealer("keys-live-test-secret-0123456789abcdef")
	ctx := context.Background()
	c := &clock{at: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}

	ks := &keySet{store: st, sealer: sealer, rotation: 90 * 24 * time.Hour, now: c.now}
	if err := ks.maintain(ctx); err != nil {
		t.Fatal(err)
	}

	first, ok := ks.signing(jose.RS256)
	if !ok {
		t.Fatal("no RS256 key on a fresh installation")
	}
	for _, alg := range jose.Algorithms {
		if s := states(ks, alg); len(s) != 1 || count(s, "signing") != 1 {
			t.Errorf("%s keys at start = %v, want one signing", alg, s)
		}
	}

	// Running again changes nothing.
	if err := ks.maintain(ctx); err != nil {
		t.Fatal(err)
	}
	if s := states(ks, jose.RS256); len(s) != 1 {
		t.Errorf("keys after maintaining twice = %v", s)
	}

	// Ninety days on, the next key is made and published, but does not sign.
	c.advance(90 * 24 * time.Hour)
	if err := ks.maintain(ctx); err != nil {
		t.Fatal(err)
	}
	if s := states(ks, jose.RS256); count(s, "signing") != 1 || count(s, "next") != 1 {
		t.Fatalf("keys when rotation is due = %v, want signing and next", s)
	}
	if key, _ := ks.signing(jose.RS256); key.ID != first.ID {
		t.Error("the new key signs before its lead")
	}
	jwks, _ := ks.public(ctx)
	if len(jwks) != 6 {
		t.Errorf("JWKS has %d keys during the lead, want both keys of each algorithm", len(jwks))
	}

	// A second server deciding at the same moment makes no third key.
	other := &keySet{store: st, sealer: sealer, rotation: 90 * 24 * time.Hour, now: c.now}
	if err := other.maintain(ctx); err != nil {
		t.Fatal(err)
	}
	if s := states(other, jose.RS256); len(s) != 2 {
		t.Errorf("keys after a second server maintained = %v, want still two", s)
	}

	// After the lead the new key signs and the old one is retired, still
	// published so tokens it signed verify.
	c.advance(keyPublishLead)
	if err := ks.maintain(ctx); err != nil {
		t.Fatal(err)
	}
	second, _ := ks.signing(jose.RS256)
	if second.ID == first.ID {
		t.Fatal("the old key still signs after the lead")
	}
	if s := states(ks, jose.RS256); s[first.ID] != "retired" || s[second.ID] != "signing" {
		t.Errorf("keys after the lead = %v", s)
	}
	if _, ok := ks.lookup(ctx)(first.ID); !ok {
		t.Error("a retired key no longer verifies")
	}

	// After the retention it is gone.
	c.advance(keyRetention + time.Minute)
	if err := ks.maintain(ctx); err != nil {
		t.Fatal(err)
	}
	if s := states(ks, jose.RS256); len(s) != 1 || s[second.ID] != "signing" {
		t.Errorf("keys after the retention = %v, want only the signing key", s)
	}

	// An emergency: rotate now and revoke the old keys. The new key signs at
	// once, and the old one is not published at all.
	if err := ks.Rotate(ctx, true, true); err != nil {
		t.Fatal(err)
	}
	third, _ := ks.signing(jose.RS256)
	if third.ID == second.ID {
		t.Fatal("an immediate rotation left the old key signing")
	}
	if s := states(ks, jose.RS256); len(s) != 1 || s[third.ID] != "signing" {
		t.Errorf("keys after revoking = %v, want only the new key, signing", s)
	}
	if _, ok := ks.lookup(ctx)(second.ID); ok {
		t.Error("a revoked key still verifies")
	}

	// A scheduled manual rotation waits the lead like an automatic one.
	if err := ks.Rotate(ctx, false, false); err != nil {
		t.Fatal(err)
	}
	if key, _ := ks.signing(jose.RS256); key.ID != third.ID {
		t.Error("a scheduled rotation signed before the lead")
	}
}
