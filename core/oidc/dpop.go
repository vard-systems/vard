package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"

	"vard/internal/domain/model"
	"vard/internal/jose"
)

// DPoP (RFC 9449): a client proves, with each request, that it holds a key,
// and the tokens it gets are bound to that key. An access token carries the
// key's thumbprint in cnf.jkt and is accepted, at userinfo, only with a proof
// from that key; a public client's refresh tokens need one too. A copied token
// is no use without the key.

const (
	ErrInvalidDPoPProof = "invalid_dpop_proof"
	TokenTypeDPoP       = "DPoP"

	dpopProofType = "dpop+jwt"
	maxDPoPJTI    = 256
)

// dpopAlgorithms are the proof signatures accepted, as the discovery
// document lists them.
var dpopAlgorithms = []string{jose.ES256, jose.RS256, jose.PS256}

// Presentation is how a protected request carried its token: the scheme of
// its Authorization header, its DPoP proof if any, and what it was for.
type Presentation struct {
	Scheme string
	Proof  string
	Method string
	URL    string
}

type dpopClaims struct {
	JTI         string      `json:"jti"`
	Method      string      `json:"htm"`
	Target      string      `json:"htu"`
	IssuedAt    json.Number `json:"iat"`
	AccessToken string      `json:"ath"`
}

// checkDPoP verifies a proof for one request (RFC 9449 §4.3) and returns its
// key's thumbprint. accessToken is the token it must name (ath), "" at the
// token endpoint. A proof is accepted once.
func (s *Service) checkDPoP(ctx context.Context, proof, method, target, accessToken string) (string, error) {
	refused := func(why string) (string, error) {
		return "", oauthError(ErrInvalidDPoPProof, why)
	}

	if proof == "" || strings.Contains(proof, ",") {
		return refused("one DPoP proof is required")
	}

	var claims dpopClaims
	header, err := jose.VerifyEmbedded(proof, &claims)
	switch {
	case err != nil:
		return refused("the DPoP proof's signature does not verify against its own key")
	case header.Type != dpopProofType:
		return refused("the DPoP proof's typ must be dpop+jwt")
	case !slices.Contains(dpopAlgorithms, header.Algorithm):
		return refused("the DPoP proof is signed with an algorithm not offered")
	case claims.JTI == "" || len(claims.JTI) > maxDPoPJTI:
		return refused("the DPoP proof needs a jti")
	case claims.Method != method:
		return refused("the DPoP proof is for another method")
	case !sameTarget(claims.Target, target):
		return refused("the DPoP proof is for another address")
	}

	issued, err := claims.IssuedAt.Int64()
	if err != nil {
		return refused("the DPoP proof needs an iat")
	}
	if now := s.now(); time.Unix(issued, 0).Before(now.Add(-model.DPoPProofWindow)) || time.Unix(issued, 0).After(now.Add(model.DPoPProofWindow)) {
		return refused("the DPoP proof was not made just now")
	}

	if accessToken != "" {
		sum := sha256.Sum256([]byte(accessToken))
		if claims.AccessToken != base64.RawURLEncoding.EncodeToString(sum[:]) {
			return refused("the DPoP proof is for another access token")
		}
	}

	thumbprint, err := jose.Thumbprint(*header.JWK)
	if err != nil {
		return refused("the DPoP proof's key is not one offered")
	}

	seen := sha256.Sum256([]byte(thumbprint + " " + claims.JTI))
	fresh, err := s.store.UseDPoPProof(ctx, hex.EncodeToString(seen[:]), s.now().Add(2*model.DPoPProofWindow))
	if err != nil {
		return "", err
	}
	if !fresh {
		return refused("the DPoP proof has already been used")
	}

	return thumbprint, nil
}

// isThumbprint says whether a value has the shape of a JWK SHA-256
// thumbprint (RFC 7638): 43 characters of unpadded base64url.
func isThumbprint(value string) bool {
	if len(value) != 43 {
		return false
	}
	sum, err := base64.RawURLEncoding.Strict().DecodeString(value)
	return err == nil && len(sum) == sha256.Size
}

// presented checks that a token bound to a key came with a proof from it,
// and that one not bound to a key did not come as DPoP.
func (s *Service) presented(ctx context.Context, token string, raw map[string]any, how Presentation) error {
	bound := boundKey(raw)

	switch {
	case bound == "" && strings.EqualFold(how.Scheme, TokenTypeDPoP):
		return oauthError(ErrInvalidToken, "the access token is not bound to a DPoP key")
	case bound == "":
		return nil
	case !strings.EqualFold(how.Scheme, TokenTypeDPoP):
		return oauthError(ErrInvalidToken, "the access token is bound to a DPoP key: send it as DPoP, with a proof")
	}

	thumbprint, err := s.checkDPoP(ctx, how.Proof, how.Method, how.URL, token)
	if err != nil {
		return err
	}
	if thumbprint != bound {
		return oauthError(ErrInvalidDPoPProof, "the DPoP proof is from another key than the token is bound to")
	}

	return nil
}

// boundKey is the thumbprint an access token is bound to, "" when it is not.
func boundKey(raw map[string]any) string {
	cnf, _ := raw["cnf"].(map[string]any)
	jkt, _ := cnf["jkt"].(string)
	return jkt
}

// sameTarget compares the address a proof names with the one it was sent to,
// without query or fragment (RFC 9449 §4.3, step 9).
func sameTarget(claimed, actual string) bool {
	a, errA := url.Parse(claimed)
	b, errB := url.Parse(actual)
	if errA != nil || errB != nil {
		return false
	}

	return strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host) && a.Path == b.Path
}
