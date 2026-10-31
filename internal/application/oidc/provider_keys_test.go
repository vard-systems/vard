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

	"vard/internal/jose"
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

// Anyone can send a token that does not verify, and each such token used to
// make the server fetch the provider's keys again. Now a forced fetch
// happens at most once a minute per key set.
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
	forged, err := jose.Sign(signingKey(t, "known"), "JWT", map[string]any{"sub": "mallory"})
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
	forged, err := jose.Sign(signingKey(t, "known"), "JWT", map[string]any{"sub": "mallory"})
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
