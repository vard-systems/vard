// Package jose signs and verifies this server's JWTs and publishes its keys. It
// is deliberately small: only RS256, PS256 and ES256 with our own keys, no
// "none", no HMAC, and a token's header never chooses the key.
package jose

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	RS256 = "RS256"
	PS256 = "PS256"
	ES256 = "ES256"
)

var Algorithms = []string{RS256, PS256, ES256}

var ErrInvalid = errors.New("invalid token")

var b64 = base64.RawURLEncoding

type Key struct {
	ID        string
	Algorithm string
	Private   crypto.Signer
	PublicKey crypto.PublicKey
}

func (k Key) public() crypto.PublicKey {
	if k.Private != nil {
		return k.Private.Public()
	}

	return k.PublicKey
}

type Header struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid,omitempty"`
	Type      string `json:"typ,omitempty"`
	// JWK is the public key a self-signed token carries (a DPoP proof).
	JWK *JWK `json:"jwk,omitempty"`
	// Critical names extensions a verifier must understand (RFC 7515
	// §4.1.11); this package understands none, so any refuses the token.
	Critical []string `json:"crit,omitempty"`
}

func Generate(algorithm string) (crypto.Signer, error) {
	switch algorithm {
	case RS256, PS256:
		return rsa.GenerateKey(rand.Reader, 2048)
	case ES256:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	default:
		return nil, fmt.Errorf("jose: unsupported algorithm %q", algorithm)
	}
}

func Sign(key Key, typ string, claims any) (string, error) {
	header, err := json.Marshal(Header{Algorithm: key.Algorithm, KeyID: key.ID, Type: typ})
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	input := b64.EncodeToString(header) + "." + b64.EncodeToString(payload)

	signature, err := sign(key, []byte(input))
	if err != nil {
		return "", err
	}

	return input + "." + b64.EncodeToString(signature), nil
}

func sign(key Key, input []byte) ([]byte, error) {
	digest := sha256.Sum256(input)

	switch key.Algorithm {
	case RS256:
		private, ok := key.Private.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("jose: %s needs an RSA key", key.Algorithm)
		}
		return rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, digest[:])
	case PS256:
		private, ok := key.Private.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("jose: %s needs an RSA key", key.Algorithm)
		}
		return rsa.SignPSS(rand.Reader, private, crypto.SHA256, digest[:], &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash})
	case ES256:
		private, ok := key.Private.(*ecdsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("jose: %s needs an EC key", key.Algorithm)
		}
		r, s, err := ecdsa.Sign(rand.Reader, private, digest[:])
		if err != nil {
			return nil, err
		}
		// JWS wants r and s as two fixed 32-byte numbers side by side, not
		// the ASN.1 structure Go's SignASN1 would make.
		out := make([]byte, 64)
		r.FillBytes(out[:32])
		s.FillBytes(out[32:])
		return out, nil
	default:
		return nil, fmt.Errorf("jose: unsupported algorithm %q", key.Algorithm)
	}
}

// Verify checks a compact JWS against the key `lookup` returns for its kid and
// decodes the claims. The key decides the algorithm, so a header cannot
// downgrade the check. Claims (issuer, expiry, audience) are the caller's to
// check.
func Verify(token string, lookup func(kid string) (Key, bool), claims any) (Header, error) {
	return VerifyAmong(token, func(header Header) []Key {
		if key, ok := lookup(header.KeyID); ok {
			return []Key{key}
		}
		return nil
	}, claims)
}

// VerifyAmong is Verify for another issuer's published keys, where a token
// without a kid may have been signed by any of them: it is accepted if one
// of the candidates of the header's algorithm verifies it. Trying several is
// no weaker than one, since each is a key the issuer published.
func VerifyAmong(token string, candidates func(Header) []Key, claims any) (Header, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Header{}, ErrInvalid
	}

	rawHeader, err := b64.DecodeString(parts[0])
	if err != nil {
		return Header{}, ErrInvalid
	}

	var header Header
	if err := json.Unmarshal(rawHeader, &header); err != nil || header.Critical != nil {
		return Header{}, ErrInvalid
	}

	signature, err := b64.DecodeString(parts[2])
	if err != nil {
		return Header{}, ErrInvalid
	}

	verified := false
	for _, key := range candidates(header) {
		if key.Algorithm == header.Algorithm && verify(key, []byte(parts[0]+"."+parts[1]), signature) {
			verified = true
			break
		}
	}
	if !verified {
		return Header{}, ErrInvalid
	}

	payload, err := b64.DecodeString(parts[1])
	if err != nil {
		return Header{}, ErrInvalid
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(claims); err != nil {
		return Header{}, ErrInvalid
	}

	return header, nil
}

func verify(key Key, input, signature []byte) bool {
	digest := sha256.Sum256(input)

	switch public := key.public().(type) {
	case *rsa.PublicKey:
		switch key.Algorithm {
		case RS256:
			return rsa.VerifyPKCS1v15(public, crypto.SHA256, digest[:], signature) == nil
		case PS256:
			return rsa.VerifyPSS(public, crypto.SHA256, digest[:], signature, &rsa.PSSOptions{SaltLength: rsa.PSSSaltLengthEqualsHash}) == nil
		}
	case *ecdsa.PublicKey:
		if key.Algorithm != ES256 || len(signature) != 64 {
			return false
		}
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:])
		return ecdsa.Verify(public, digest[:], r, s)
	}

	return false
}

// JWK is a public key as a JSON Web Key (RFC 7517).
type JWK struct {
	KeyType   string `json:"kty"`
	Use       string `json:"use"`
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`

	// RSA
	N string `json:"n,omitempty"`
	E string `json:"e,omitempty"`

	// EC
	Curve string `json:"crv,omitempty"`
	X     string `json:"x,omitempty"`
	Y     string `json:"y,omitempty"`
}

func (k Key) Public() (JWK, error) {
	jwk := JWK{Use: "sig", Algorithm: k.Algorithm, KeyID: k.ID}

	switch public := k.Private.Public().(type) {
	case *rsa.PublicKey:
		jwk.KeyType = "RSA"
		jwk.N = b64.EncodeToString(public.N.Bytes())
		jwk.E = b64.EncodeToString(big.NewInt(int64(public.E)).Bytes())
	case *ecdsa.PublicKey:
		point, err := public.Bytes()
		if err != nil {
			return JWK{}, err
		}
		jwk.KeyType = "EC"
		jwk.Curve = "P-256"
		jwk.X = b64.EncodeToString(point[1:33])
		jwk.Y = b64.EncodeToString(point[33:65])
	default:
		return JWK{}, fmt.Errorf("jose: unsupported key type %T", public)
	}

	return jwk, nil
}

func ParseJWK(jwk JWK) (Key, error) {
	key := Key{ID: jwk.KeyID, Algorithm: jwk.Algorithm}

	switch jwk.KeyType {
	case "RSA":
		n, err := b64.DecodeString(jwk.N)
		if err != nil {
			return Key{}, fmt.Errorf("jose: the key's modulus is not base64url: %w", err)
		}
		e, err := b64.DecodeString(jwk.E)
		if err != nil || len(e) == 0 || len(e) > 4 {
			return Key{}, fmt.Errorf("jose: the key's exponent is not usable")
		}
		if len(n) < 256 {
			return Key{}, fmt.Errorf("jose: an RSA key shorter than 2048 bits is not trusted")
		}

		key.PublicKey = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
		if key.Algorithm == "" {
			key.Algorithm = RS256
		}
	case "EC":
		if jwk.Curve != "P-256" {
			return Key{}, fmt.Errorf("jose: the curve %q is not supported", jwk.Curve)
		}
		x, errX := b64.DecodeString(jwk.X)
		y, errY := b64.DecodeString(jwk.Y)
		if errX != nil || errY != nil || len(x) != 32 || len(y) != 32 {
			return Key{}, fmt.Errorf("jose: the key's point is not usable")
		}

		public, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), append(append([]byte{4}, x...), y...))
		if err != nil {
			return Key{}, fmt.Errorf("jose: the key's point is not on the curve: %w", err)
		}
		key.PublicKey = public
		if key.Algorithm == "" {
			key.Algorithm = ES256
		}
	default:
		return Key{}, fmt.Errorf("jose: the key type %q is not supported", jwk.KeyType)
	}

	return key, nil
}

// VerifyEmbedded checks a token signed with the key its own header carries, as
// a DPoP proof is (RFC 9449 §4.2), and returns the header with that key. It
// proves only that whoever made the token holds the key; what the key is
// trusted for is the caller's to decide.
func VerifyEmbedded(token string, claims any) (Header, error) {
	var parsed Key
	header, err := VerifyAmong(token, func(header Header) []Key {
		if header.JWK == nil {
			return nil
		}
		key, err := ParseJWK(*header.JWK)
		if err != nil {
			return nil
		}
		// The header names the algorithm; verify refuses one the key cannot do.
		key.Algorithm = header.Algorithm
		parsed = key
		return []Key{key}
	}, claims)
	if err != nil || parsed.PublicKey == nil {
		return Header{}, ErrInvalid
	}

	return header, nil
}

// Thumbprint is a public key's JWK SHA-256 thumbprint (RFC 7638): its
// required members only, in order, without whitespace.
func Thumbprint(jwk JWK) (string, error) {
	var canonical string
	switch jwk.KeyType {
	case "EC":
		canonical = fmt.Sprintf(`{"crv":%q,"kty":"EC","x":%q,"y":%q}`, jwk.Curve, jwk.X, jwk.Y)
	case "RSA":
		canonical = fmt.Sprintf(`{"e":%q,"kty":"RSA","n":%q}`, jwk.E, jwk.N)
	default:
		return "", fmt.Errorf("jose: the key type %q has no thumbprint here", jwk.KeyType)
	}

	sum := sha256.Sum256([]byte(canonical))
	return b64.EncodeToString(sum[:]), nil
}

// HalfHash is at_hash (OpenID Connect Core 3.1.3.6): the base64url left half of
// a SHA-256.
func HalfHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return b64.EncodeToString(sum[:len(sum)/2])
}
