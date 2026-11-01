package oidc

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"vard/tools/jose"
)

// publishedKeys stands in for a provider's JWKS endpoint: it answers every
// request with the keys it is given and counts the requests.
type publishedKeys struct {
	mu      sync.Mutex
	keys    []jose.Key
	fetches int
}

func (p *publishedKeys) RoundTrip(*http.Request) (*http.Response, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fetches++

	set := struct {
		Keys []jose.JWK `json:"keys"`
	}{}
	for _, key := range p.keys {
		jwk, err := key.Public()
		if err != nil {
			return nil, err
		}
		set.Keys = append(set.Keys, jwk)
	}
	body, err := json.Marshal(set)
	if err != nil {
		return nil, err
	}

	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(body))), Header: http.Header{}}, nil
}

func (p *publishedKeys) publish(keys ...jose.Key) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.keys = keys
}

func (p *publishedKeys) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.fetches
}

func signingKey(t *testing.T, kid string) jose.Key {
	t.Helper()

	private, err := jose.Generate(jose.ES256)
	if err != nil {
		t.Fatal(err)
	}
	return jose.Key{ID: kid, Algorithm: jose.ES256, Private: private}
}

func serviceWithKeys(published *publishedKeys, c *clock) *Service {
	return &Service{
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		social:   &http.Client{Transport: published},
		ssoCache: newSSOCache(),
		now:      c.now,
	}
}

const testJWKS = "https://idp.example/jwks"

// Anyone can send a token naming a key the server does not hold, and each
// such token used to make it fetch the provider's keys again. Now a forced
// fetch happens at most once a minute per key set.
func TestForcedKeyRefetchesAreThrottled(t *testing.T) {
	c := &clock{at: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	known := signingKey(t, "known")
	published := &publishedKeys{keys: []jose.Key{known}}
	s := serviceWithKeys(published, c)
	ctx := context.Background()

	good, err := jose.Sign(known, "JWT", map[string]any{"sub": "ada"})
	if err != nil {
		t.Fatal(err)
	}
	forged, err := jose.Sign(signingKey(t, "invented"), "JWT", map[string]any{"sub": "mallory"})
	if err != nil {
		t.Fatal(err)
	}

	var claims map[string]any
	if err := s.verifyPublished(ctx, testJWKS, good, &claims); err != nil {
		t.Fatalf("a token the provider signed = %v", err)
	}
	if got := published.count(); got != 1 {
		t.Fatalf("fetches after the first token = %d, want 1", got)
	}

	for i := range 2 {
		if err := s.verifyPublished(ctx, testJWKS, forged, &claims); err == nil {
			t.Fatalf("forged token %d verified", i+1)
		}
		c.advance(10 * time.Second)
	}
	if got := published.count(); got != 2 {
		t.Errorf("fetches after two forged tokens in a row = %d, want 2 (one forced)", got)
	}

	c.advance(keyRefetchEvery)
	if err := s.verifyPublished(ctx, testJWKS, forged, &claims); err == nil {
		t.Fatal("a forged token verified after the window")
	}
	if got := published.count(); got != 3 {
		t.Errorf("fetches once the minute has passed = %d, want 3", got)
	}

	// The keys held still verify what the provider signs while forced
	// fetches are held back.
	if err := s.verifyPublished(ctx, testJWKS, good, &claims); err != nil {
		t.Errorf("a good token inside the window = %v", err)
	}
}

// Forged tokens arriving together still make one fetch between them.
func TestConcurrentFailuresShareOneForcedFetch(t *testing.T) {
	c := &clock{at: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)}
	known := signingKey(t, "known")
	published := &publishedKeys{keys: []jose.Key{known}}
	s := serviceWithKeys(published, c)
	ctx := context.Background()

	if _, err := s.providerKeys(ctx, testJWKS, false); err != nil {
		t.Fatal(err)
	}
	forged, err := jose.Sign(signingKey(t, "invented"), "JWT", map[string]any{"sub": "mallory"})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			var claims map[string]any
			_ = s.verifyPublished(ctx, testJWKS, forged, &claims)
		})
	}
	wg.Wait()

	if got := published.count(); got != 2 {
		t.Errorf("fetches = %d, want 2 (the first and one forced)", got)
	}
}

// Only a key the server does not hold is worth fetching the keys again for.
// A token naming a key it holds, with a signature that key does not make, is
// refused without a fetch, so it cannot spend the refetch a real rotation
// would need; a token naming no key may be signed with any, so it may.
func TestOnlyAnUnknownKeyForcesAFetch(t *testing.T) {
	known := signingKey(t, "known")
	sign := func(key jose.Key) string {
		token, err := jose.Sign(key, "JWT", map[string]any{"sub": "mallory"})
		if err != nil {
			t.Fatal(err)
		}
		return token
	}

	tests := []struct {
		name    string
		token   string
		fetches int
	}{
		{name: "a known kid with a bad signature", token: sign(signingKey(t, "known")), fetches: 1},
		{name: "an unknown kid", token: sign(signingKey(t, "invented")), fetches: 2},
		{name: "no kid at all", token: sign(signingKey(t, "")), fetches: 2},
		{name: "not a token", token: "not.a-token", fetches: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			published := &publishedKeys{keys: []jose.Key{known}}
			s := serviceWithKeys(published, &clock{at: time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)})
			if _, err := s.providerKeys(context.Background(), testJWKS, false); err != nil {
				t.Fatal(err)
			}

			var claims map[string]any
			if err := s.verifyPublished(context.Background(), testJWKS, tt.token, &claims); err == nil {
				t.Fatal("the token verified")
			}
			if got := published.count(); got != tt.fetches {
				t.Errorf("fetches = %d, want %d", got, tt.fetches)
			}
		})
	}
}
