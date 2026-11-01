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

// formPost posts a form to a provider endpoint as a client: HTTP Basic when
// a secret is given, the client id in the form otherwise, and a DPoP proof
// when one is given.
func formPost(t *testing.T, f *oauthFixture, path, clientID, secret, proof string, form url.Values) tokenResult {
	t.Helper()

	if secret == "" {
		form.Set("client_id", clientID)
	}
	req, _ := http.NewRequest(http.MethodPost, f.s.root+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if secret != "" {
		req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(secret))
	}
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

// A code asked for with dpop_jkt (RFC 9449 §10) is redeemed only with a
// proof from that key, so a code intercepted on the way back is no use to
// whoever lacks the key. Tried without it, the code is spent, as it is when
// its PKCE verifier is wrong.
func TestLiveDPoPBindsTheAuthorizationCode(t *testing.T) {
	f := newOAuthFixture(t)
	spaID, _, _ := f.register(map[string]any{
		"name": "Single page", "type": "spa",
		"grant_types":   []string{"authorization_code", "refresh_token"},
		"redirect_uris": []string{shopRedirect},
		"scopes":        []string{"openid", "email", "offline_access"},
	})
	f.clientID, f.clientSecret = spaID, ""

	key, other := newDPoPKey(t), newDPoPKey(t)
	tokenURL := f.s.cfg.Issuer + "/oauth2/token"

	boundCode := func() (code, verifier string) {
		verifier, challenge := pkce(t)
		back := f.signInWith(f.s.browser(), challenge, url.Values{
			"scope": {"openid email offline_access"}, "audience": {""}, "dpop_jkt": {key.thumbprint(t)},
		})
		if back.Query().Get("code") == "" {
			t.Fatalf("authorize with dpop_jkt came back with %s", back)
		}
		return back.Query().Get("code"), verifier
	}
	exchange := func(code, verifier, proof string) tokenResult {
		return postToken(t, f, spaID, proof, url.Values{
			"grant_type": {"authorization_code"}, "code": {code},
			"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
		})
	}

	t.Run("a proof from the named key gets tokens bound to it", func(t *testing.T) {
		code, verifier := boundCode()
		tokens := exchange(code, verifier, key.proof(t, "POST", tokenURL, ""))
		if tokens.status != http.StatusOK || tokens.str("token_type") != "DPoP" {
			t.Fatalf("= %d %v", tokens.status, tokens.body)
		}
		if _, claims := f.verifyJWT(tokens.str("access_token")); claims["cnf"].(map[string]any)["jkt"] != key.thumbprint(t) {
			t.Errorf("cnf = %v, want the named key's thumbprint", claims["cnf"])
		}
	})

	for _, tt := range []struct {
		name  string
		proof func() string
	}{
		{"a proof from another key is refused", func() string { return other.proof(t, "POST", tokenURL, "") }},
		{"no proof at all is refused", func() string { return "" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			code, verifier := boundCode()
			if refused := exchange(code, verifier, tt.proof()); refused.status != http.StatusBadRequest || refused.str("error") != "invalid_grant" {
				t.Fatalf("= %d %v, want invalid_grant", refused.status, refused.body)
			}
			if again := exchange(code, verifier, key.proof(t, "POST", tokenURL, "")); again.str("error") != "invalid_grant" {
				t.Errorf("the code with the right key after a refusal = %d %v, want it spent", again.status, again.body)
			}
		})
	}

	t.Run("a malformed dpop_jkt goes back to the client", func(t *testing.T) {
		_, challenge := pkce(t)
		for _, bad := range []string{"short", strings.Repeat("A", 42) + "!", strings.Repeat("A", 44)} {
			back := f.s.browser().visit(f.authorizeURL(challenge, url.Values{"scope": {"openid"}, "audience": {""}, "dpop_jkt": {bad}}))
			if !strings.HasPrefix(back.String(), shopRedirect) || back.Query().Get("error") != "invalid_request" {
				t.Errorf("dpop_jkt %q led to %s, want invalid_request at the client", bad, back)
			}
		}
	})
}

// A pushed request that brings a DPoP proof binds its code to the proof's key
// (RFC 9449 §10.1), with no dpop_jkt needed; one whose dpop_jkt names another
// key than its proof is refused.
func TestLiveDPoPBindsAPushedRequest(t *testing.T) {
	f := newOAuthFixture(t)
	key, other := newDPoPKey(t), newDPoPKey(t)
	parURL, tokenURL := f.s.cfg.Issuer+"/oauth2/par", f.s.cfg.Issuer+"/oauth2/token"

	push := func(proof string, extra url.Values) (tokenResult, string) {
		verifier, challenge := pkce(t)
		form := url.Values{
			"response_type": {"code"}, "redirect_uri": {shopRedirect}, "scope": {"openid email"},
			"code_challenge": {challenge}, "code_challenge_method": {"S256"},
		}
		for name, values := range extra {
			form[name] = values
		}
		return formPost(t, f, "/oauth2/par", f.clientID, f.clientSecret, proof, form), verifier
	}
	codeFor := func(requestURI string) string {
		b := f.s.browser()
		login := b.visit(f.s.root + "/oauth2/authorize?" + url.Values{"client_id": {f.clientID}, "request_uri": {requestURI}}.Encode())
		var out struct {
			RedirectTo string `json:"redirect_to"`
		}
		b.account(http.MethodPost, "/login", map[string]string{
			"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
		}, &out)
		back, _ := url.Parse(out.RedirectTo)
		if back.Query().Get("code") == "" {
			t.Fatalf("the pushed request came back with %s", out.RedirectTo)
		}
		return back.Query().Get("code")
	}
	exchange := func(code, verifier, proof string) tokenResult {
		return formPost(t, f, "/oauth2/token", f.clientID, f.clientSecret, proof, url.Values{
			"grant_type": {"authorization_code"}, "code": {code},
			"redirect_uri": {shopRedirect}, "code_verifier": {verifier},
		})
	}

	t.Run("the push's proof binds the code without dpop_jkt", func(t *testing.T) {
		pushed, verifier := push(key.proof(t, "POST", parURL, ""), nil)
		if pushed.status != http.StatusCreated {
			t.Fatalf("pushing with a proof = %d %v", pushed.status, pushed.body)
		}
		if refused := exchange(codeFor(pushed.str("request_uri")), verifier, ""); refused.str("error") != "invalid_grant" {
			t.Errorf("its code without a proof = %d %v, want invalid_grant", refused.status, refused.body)
		}

		pushed, verifier = push(key.proof(t, "POST", parURL, ""), nil)
		tokens := exchange(codeFor(pushed.str("request_uri")), verifier, key.proof(t, "POST", tokenURL, ""))
		if tokens.status != http.StatusOK || tokens.str("token_type") != "DPoP" {
			t.Fatalf("its code with a proof from the key = %d %v", tokens.status, tokens.body)
		}
	})

	t.Run("a proof and a dpop_jkt that agree are taken", func(t *testing.T) {
		if pushed, _ := push(key.proof(t, "POST", parURL, ""), url.Values{"dpop_jkt": {key.thumbprint(t)}}); pushed.status != http.StatusCreated {
			t.Errorf("= %d %v, want 201", pushed.status, pushed.body)
		}
	})

	t.Run("a proof and a dpop_jkt that disagree are refused", func(t *testing.T) {
		refused, _ := push(key.proof(t, "POST", parURL, ""), url.Values{"dpop_jkt": {other.thumbprint(t)}})
		if refused.status != http.StatusBadRequest || refused.str("error") != "invalid_dpop_proof" {
			t.Errorf("= %d %v, want invalid_dpop_proof", refused.status, refused.body)
		}
	})

	t.Run("a malformed dpop_jkt is the client's to read", func(t *testing.T) {
		refused, _ := push("", url.Values{"dpop_jkt": {"not-a-thumbprint"}})
		if refused.status != http.StatusBadRequest || refused.str("error") != "invalid_request" {
			t.Errorf("= %d %v, want invalid_request", refused.status, refused.body)
		}
	})

	t.Run("a proof for another address is refused", func(t *testing.T) {
		refused, _ := push(key.proof(t, "POST", tokenURL, ""), nil)
		if refused.str("error") != "invalid_dpop_proof" {
			t.Errorf("= %d %v, want invalid_dpop_proof", refused.status, refused.body)
		}
	})
}
