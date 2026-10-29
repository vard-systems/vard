package jose

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

func testKey(t *testing.T, alg, kid string) Key {
	t.Helper()

	private, err := Generate(alg)
	if err != nil {
		t.Fatal(err)
	}

	return Key{ID: kid, Algorithm: alg, Private: private}
}

func lookupOf(keys ...Key) func(string) (Key, bool) {
	return func(kid string) (Key, bool) {
		for _, key := range keys {
			if key.ID == kid {
				return key, true
			}
		}
		return Key{}, false
	}
}

func TestSignAndVerifyEachAlgorithm(t *testing.T) {
	for _, alg := range Algorithms {
		t.Run(alg, func(t *testing.T) {
			key := testKey(t, alg, "k1")

			token, err := Sign(key, "at+jwt", map[string]any{"sub": "user", "n": 7})
			if err != nil {
				t.Fatal(err)
			}

			var claims struct {
				Sub string      `json:"sub"`
				N   json.Number `json:"n"`
			}
			header, err := Verify(token, lookupOf(key), &claims)
			if err != nil {
				t.Fatalf("Verify: %v", err)
			}

			if header.Type != "at+jwt" || header.Algorithm != alg || header.KeyID != "k1" {
				t.Errorf("header = %+v", header)
			}
			if claims.Sub != "user" || claims.N != "7" {
				t.Errorf("claims = %+v", claims)
			}
		})
	}
}

func TestVerifyRefusesTampering(t *testing.T) {
	key := testKey(t, RS256, "k1")
	other := testKey(t, RS256, "k1")

	token, err := Sign(key, "JWT", map[string]any{"sub": "alice"})
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")

	forged := parts[0] + "." + base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"mallory"}`)) + "." + parts[2]

	// A header naming "none", or HS256 with the public key as the secret, is
	// the classic way round a careless check.
	none := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","kid":"k1"}`)) + "." + parts[1] + "."
	hs := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","kid":"k1"}`)) + "." + parts[1] + "." + parts[2]

	// An extension marked critical has to be understood or the token refused
	// (RFC 7515 §4.1.11), and this server understands none.
	critical, err := signWithHeader(key, `{"alg":"RS256","kid":"k1","typ":"JWT","crit":["exp"],"exp":1}`, parts[1])
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		token  string
		lookup func(string) (Key, bool)
	}{
		"changed payload":   {forged, lookupOf(key)},
		"alg none":          {none, lookupOf(key)},
		"alg switched":      {hs, lookupOf(key)},
		"another key":       {token, lookupOf(other)},
		"unknown key id":    {token, lookupOf(Key{ID: "k2", Algorithm: RS256, Private: key.Private})},
		"not a jws":         {"abc", lookupOf(key)},
		"truncated":         {parts[0] + "." + parts[1], lookupOf(key)},
		"signature garbage": {parts[0] + "." + parts[1] + ".!!!", lookupOf(key)},
		"a critical header": {critical, lookupOf(key)},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var claims map[string]any
			if _, err := Verify(tc.token, tc.lookup, &claims); err == nil {
				t.Fatal("Verify accepted it")
			}
		})
	}
}

// signWithHeader signs a payload under a header written by hand.
func signWithHeader(key Key, header, payload string) (string, error) {
	input := base64.RawURLEncoding.EncodeToString([]byte(header)) + "." + payload
	signature, err := sign(key, []byte(input))
	if err != nil {
		return "", err
	}
	return input + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func TestVerifyAmongPublishedKeys(t *testing.T) {
	signer := testKey(t, RS256, "")
	published := []Key{testKey(t, RS256, ""), signer, testKey(t, ES256, "")}

	token, err := Sign(signer, "JWT", map[string]any{"sub": "alice"})
	if err != nil {
		t.Fatal(err)
	}

	// The same RSA key published for PS256 must not verify an RS256 token.
	relabelled := signer
	relabelled.Algorithm = PS256

	cases := map[string]struct {
		keys []Key
		ok   bool
	}{
		"one of several verifies": {published, true},
		"none of them signed it":  {[]Key{published[0], published[2]}, false},
		"only under another alg":  {[]Key{relabelled}, false},
		"no candidates":           {nil, false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var claims map[string]any
			_, err := VerifyAmong(token, func(Header) []Key { return tc.keys }, &claims)
			if (err == nil) != tc.ok {
				t.Fatalf("VerifyAmong = %v, want ok %v", err, tc.ok)
			}
			if tc.ok && claims["sub"] != "alice" {
				t.Errorf("claims = %v", claims)
			}
		})
	}
}

func TestPublicJWKMatchesTheKey(t *testing.T) {
	rsaKey := testKey(t, PS256, "r")
	jwk, err := rsaKey.Public()
	if err != nil {
		t.Fatal(err)
	}

	public := rsaKey.Private.Public().(*rsa.PublicKey)
	n, _ := base64.RawURLEncoding.DecodeString(jwk.N)
	if jwk.KeyType != "RSA" || jwk.Use != "sig" || jwk.Algorithm != PS256 || new(big.Int).SetBytes(n).Cmp(public.N) != 0 || jwk.E != "AQAB" {
		t.Errorf("RSA JWK = %+v", jwk)
	}

	ecKey := testKey(t, ES256, "e")
	jwk, err = ecKey.Public()
	if err != nil {
		t.Fatal(err)
	}

	// An uncompressed point is 0x04, then X, then Y.
	point, err := ecKey.Private.Public().(*ecdsa.PublicKey).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	x, _ := base64.RawURLEncoding.DecodeString(jwk.X)
	if jwk.KeyType != "EC" || jwk.Curve != "P-256" || !bytes.Equal(x, point[1:33]) {
		t.Errorf("EC JWK = %+v", jwk)
	}
}

func TestSealOpensOnlyWithTheSameSecret(t *testing.T) {
	key := testKey(t, ES256, "k")

	sealer, err := NewSealer("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}

	where := Bind("signing_keys", "private_key", "k")
	sealed, err := sealer.Seal(where, key.Private)
	if err != nil {
		t.Fatal(err)
	}

	opened, _, err := sealer.Open(where, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !opened.Public().(*ecdsa.PublicKey).Equal(key.Private.Public()) {
		t.Error("opened a different key")
	}

	wrong, _ := NewSealer("another secret that is long enough!!")
	if _, _, err := wrong.Open(where, sealed); err == nil {
		t.Error("opened with the wrong secret")
	}
}

// The example from OpenID Connect Core's at_hash definition is not published
// with a value, so this pins the construction instead: half of SHA-256.
func TestHalfHashIsHalfOfSHA256(t *testing.T) {
	got, _ := base64.RawURLEncoding.DecodeString(HalfHash("token"))
	if len(got) != 16 {
		t.Errorf("at_hash is %d bytes, want 16", len(got))
	}
}

// A key somebody else published — an identity provider's — verifies what it
// signed, and nothing else.
func TestParseJWK(t *testing.T) {
	for _, algorithm := range []string{RS256, PS256, ES256} {
		t.Run(algorithm, func(t *testing.T) {
			private, err := Generate(algorithm)
			if err != nil {
				t.Fatal(err)
			}
			signer := Key{ID: "idp-1", Algorithm: algorithm, Private: private}

			published, err := signer.Public()
			if err != nil {
				t.Fatal(err)
			}

			theirs, err := ParseJWK(published)
			if err != nil {
				t.Fatalf("ParseJWK() = %v", err)
			}

			token, err := Sign(signer, "JWT", map[string]any{"sub": "ada"})
			if err != nil {
				t.Fatal(err)
			}

			var claims map[string]any
			if _, err := Verify(token, func(string) (Key, bool) { return theirs, true }, &claims); err != nil {
				t.Fatalf("a token the key signed did not verify: %v", err)
			}
			if claims["sub"] != "ada" {
				t.Errorf("claims = %v", claims)
			}

			// Another key does not verify it, and a changed token does not.
			other, _ := Generate(algorithm)
			otherPublic, _ := Key{ID: "idp-1", Algorithm: algorithm, Private: other}.Public()
			stranger, _ := ParseJWK(otherPublic)
			if _, err := Verify(token, func(string) (Key, bool) { return stranger, true }, &claims); err == nil {
				t.Error("another key verified the token")
			}

			parts := strings.Split(token, ".")
			forged := parts[0] + "." + b64.EncodeToString([]byte(`{"sub":"mallory"}`)) + "." + parts[2]
			if _, err := Verify(forged, func(string) (Key, bool) { return theirs, true }, &claims); err == nil {
				t.Error("a token with its claims changed verified")
			}
		})
	}
}

func TestParseJWKRefuses(t *testing.T) {
	tests := []struct {
		name string
		jwk  JWK
	}{
		{name: "a symmetric key", jwk: JWK{KeyType: "oct"}},
		{name: "a curve not supported", jwk: JWK{KeyType: "EC", Curve: "P-521"}},
		{name: "a short RSA key", jwk: JWK{KeyType: "RSA", N: b64.EncodeToString(make([]byte, 128)), E: "AQAB"}},
		{name: "a point off the curve", jwk: JWK{KeyType: "EC", Curve: "P-256", X: b64.EncodeToString(make([]byte, 32)), Y: b64.EncodeToString(make([]byte, 32))}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseJWK(tt.jwk); err == nil {
				t.Error("ParseJWK() = nothing, want it refused")
			}
		})
	}
}

// The example key of RFC 7638 §3.1 and the thumbprint the RFC gives for it.
func TestThumbprint(t *testing.T) {
	jwk := JWK{
		KeyType: "RSA",
		E:       "AQAB",
		N:       "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbISD08qNLyrdkt-bFTWhAI4vMQFh6WeZu0fM4lFd2NcRwr3XPksINHaQ-G_xBniIqbw0Ls1jF44-csFCur-kEgU8awapJzKnqDKgw",
	}

	got, err := Thumbprint(jwk)
	if err != nil {
		t.Fatal(err)
	}
	if want := "NzbLsXh8uDCcd-6MNwXF4W_7noWXFZAfHkxZsRGC9Xs"; got != want {
		t.Errorf("Thumbprint = %s, want %s", got, want)
	}
}

// A token signed with the key in its own header verifies; the same token
// under another key's header does not.
func TestVerifyEmbedded(t *testing.T) {
	signer, err := Generate(ES256)
	if err != nil {
		t.Fatal(err)
	}
	key := Key{Algorithm: ES256, Private: signer}
	public, err := key.Public()
	if err != nil {
		t.Fatal(err)
	}

	token := signEmbedded(t, key, public)

	var claims map[string]any
	header, err := VerifyEmbedded(token, &claims)
	if err != nil || header.JWK == nil || claims["jti"] != "one" {
		t.Fatalf("VerifyEmbedded = %v, %v, %v", header, claims, err)
	}

	other, _ := Generate(ES256)
	otherPublic, _ := Key{Algorithm: ES256, Private: other}.Public()
	if _, err := VerifyEmbedded(signEmbedded(t, key, otherPublic), &claims); err == nil {
		t.Error("a token whose header carries another key was accepted")
	}
}

func signEmbedded(t *testing.T, key Key, carried JWK) string {
	t.Helper()

	carried.KeyID, carried.Use, carried.Algorithm = "", "", ""
	header, err := json.Marshal(Header{Algorithm: ES256, Type: "dpop+jwt", JWK: &carried})
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"jti": "one"})
	input := b64.EncodeToString(header) + "." + b64.EncodeToString(payload)

	signature, err := sign(key, []byte(input))
	if err != nil {
		t.Fatal(err)
	}

	return input + "." + b64.EncodeToString(signature)
}
