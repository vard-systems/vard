package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"vard/internal/jose"
)

// dpopKey is a client's key, making DPoP proofs as a client library would.
type dpopKey struct {
	key *ecdsa.PrivateKey
	jwk jose.JWK
}

func newDPoPKey(t *testing.T) *dpopKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	public, err := key.PublicKey.ECDH()
	if err != nil {
		t.Fatal(err)
	}
	point := public.Bytes()

	return &dpopKey{key: key, jwk: jose.JWK{
		KeyType: "EC", Curve: "P-256",
		X: b64.EncodeToString(point[1:33]), Y: b64.EncodeToString(point[33:65]),
	}}
}

func (k *dpopKey) thumbprint(t *testing.T) string {
	t.Helper()
	thumbprint, err := jose.Thumbprint(k.jwk)
	if err != nil {
		t.Fatal(err)
	}
	return thumbprint
}

// proof is a DPoP proof for one request; accessToken "" leaves out ath.
func (k *dpopKey) proof(t *testing.T, method, target, accessToken string) string {
	t.Helper()

	header, _ := json.Marshal(map[string]any{
		"typ": "dpop+jwt", "alg": "ES256",
		"jwk": map[string]string{"kty": "EC", "crv": "P-256", "x": k.jwk.X, "y": k.jwk.Y},
	})
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	claims := map[string]any{"jti": b64.EncodeToString(id), "htm": method, "htu": target, "iat": time.Now().Unix()}
	if accessToken != "" {
		sum := sha256.Sum256([]byte(accessToken))
		claims["ath"] = b64.EncodeToString(sum[:])
	}
	payload, _ := json.Marshal(claims)

	input := b64.EncodeToString(header) + "." + b64.EncodeToString(payload)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, k.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	return input + "." + b64.EncodeToString(signature)
}

// postToken is the token endpoint with a DPoP proof, for a public client.
func postToken(t *testing.T, f *oauthFixture, clientID, proof string, form url.Values) tokenResult {
	t.Helper()

	form.Set("client_id", clientID)
	req, _ := http.NewRequest(http.MethodPost, f.s.root+"/oauth2/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if proof != "" {
		req.Header.Set("DPoP", proof)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	result := tokenResult{status: res.StatusCode, body: map[string]any{}}
	_ = json.NewDecoder(res.Body).Decode(&result.body)
	return result
}

// A client holding a key gets tokens bound to it: an access token userinfo
// takes only as DPoP with a fresh proof from that key, and a refresh token
// that needs one too. Copied without the key, neither is any use.
func TestLiveDPoP(t *testing.T) {
	f := newOAuthFixture(t)
	spaID, _, _ := f.register(map[string]any{
		"name": "Single page", "type": "spa",
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{shopRedirect},
		"scopes":        []string{"openid", "email", "offline_access"},
	})
	f.clientID, f.clientSecret = spaID, ""

	key := newDPoPKey(t)
	tokenURL := f.s.cfg.Issuer + "/oauth2/token"
	userinfoURL := f.s.cfg.Issuer + "/oauth2/userinfo"

	b := f.s.browser()
	verifier, challenge := pkce(t)
	back := f.signInWith(b, challenge, url.Values{"scope": {"openid email offline_access"}, "audience": {""}})
	tokens := postToken(t, f, spaID, key.proof(t, "POST", tokenURL, ""), url.Values{
		"grant_type": {"authorization_code"}, "code": {back.Query().Get("code")},
		"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
	})
	if tokens.status != http.StatusOK || tokens.str("token_type") != "DPoP" {
		t.Fatalf("the code with a proof = %d %v", tokens.status, tokens.body)
	}
	access := tokens.str("access_token")
	if _, claims := f.verifyJWT(access); claims["cnf"].(map[string]any)["jkt"] != key.thumbprint(t) {
		t.Errorf("the access token's cnf is %v, want the key's thumbprint", claims["cnf"])
	}

	userinfo := func(scheme, proof string) (int, string) {
		req, _ := http.NewRequest(http.MethodGet, f.s.root+"/oauth2/userinfo", nil)
		req.Header.Set("Authorization", scheme+" "+access)
		if proof != "" {
			req.Header.Set("DPoP", proof)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		body, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(body)
	}

	proof := key.proof(t, "GET", userinfoURL, access)
	if status, body := userinfo("DPoP", proof); status != http.StatusOK {
		t.Fatalf("userinfo with the token and its proof = %d %s", status, body)
	}

	tests := []struct {
		name   string
		scheme string
		proof  string
	}{
		{"as a bearer token, without the key", "Bearer", ""},
		{"with a proof already used", "DPoP", proof},
		{"with a proof from another key", "DPoP", newDPoPKey(t).proof(t, "GET", userinfoURL, access)},
		{"with a proof for another address", "DPoP", key.proof(t, "GET", f.s.cfg.Issuer+"/elsewhere", access)},
		{"with a proof for another token", "DPoP", key.proof(t, "GET", userinfoURL, "another-token")},
	}
	for _, test := range tests {
		if status, _ := userinfo(test.scheme, test.proof); status != http.StatusUnauthorized {
			t.Errorf("userinfo %s = %d, want 401", test.name, status)
		}
	}

	refresh := tokens.str("refresh_token")
	asks := func(proof string) tokenResult {
		return postToken(t, f, spaID, proof, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}})
	}
	if got := asks(""); got.str("error") != "invalid_grant" {
		t.Errorf("refreshing without a proof = %v, want invalid_grant", got.body)
	}
	if got := asks(newDPoPKey(t).proof(t, "POST", tokenURL, "")); got.str("error") != "invalid_grant" {
		t.Errorf("refreshing with another key = %v, want invalid_grant", got.body)
	}
	if got := asks(key.proof(t, "POST", tokenURL, "")); got.status != http.StatusOK || got.str("token_type") != "DPoP" {
		t.Errorf("refreshing with the key = %d %v", got.status, got.body)
	}
}
