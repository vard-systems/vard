package api

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"

	"vard/internal/domain/totp"
)

// heldSignIn signs ada in from a fresh browser and expects the sign-in to
// wait for her second factor.
func (f *oauthFixture) heldSignIn(t *testing.T) *browser {
	t.Helper()

	b := f.s.browser()
	_, challenge := pkce(t)
	login := b.visit(f.authorizeURL(challenge, nil))

	var out struct {
		SecondFactor bool   `json:"second_factor"`
		RedirectTo   string `json:"redirect_to"`
	}
	b.account(http.MethodPost, "/login", map[string]any{
		"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
	}, &out)
	if !out.SecondFactor || out.RedirectTo != "" {
		t.Fatalf("the sign-in answered %+v, want it held for the second factor", out)
	}

	return b
}

type secondFactorOffer struct {
	SecondFactor struct {
		Methods []string       `json:"methods"`
		Passkey map[string]any `json:"passkey"`
	} `json:"second_factor"`
}

// answerSecondFactor posts a second factor and returns the status and the
// problem code or where the browser goes.
func answerSecondFactor(b *browser, body map[string]any) (int, string) {
	var out struct {
		RedirectTo string `json:"redirect_to"`
		Code       string `json:"code"`
	}
	status := b.account(http.MethodPost, "/login/second-factor", body, &out)
	if status == http.StatusOK {
		return status, out.RedirectTo
	}

	return status, out.Code
}

func codeAt(t *testing.T, secret string, step int64) string {
	t.Helper()

	code, err := totp.Code(secret, step)
	if err != nil {
		t.Fatal(err)
	}

	return code
}

// addAuthenticator sets up an authenticator for whoever b is signed in as,
// confirming it with the code of the step before this one, and returns its
// secret and the recovery codes.
func addAuthenticator(t *testing.T, b *browser) (string, []string) {
	t.Helper()

	var setup struct {
		Authenticator struct {
			Secret string `json:"secret"`
		} `json:"authenticator"`
	}
	if status := b.account(http.MethodPost, "/two-step/authenticator", map[string]any{"current_password": adaPassword}, &setup); status != http.StatusOK {
		t.Fatalf("starting an authenticator = %d", status)
	}

	var confirmed struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	code := codeAt(t, setup.Authenticator.Secret, totp.Step(time.Now())-1)
	if status := b.account(http.MethodPost, "/two-step/authenticator/confirm", map[string]any{"code": code}, &confirmed); status != http.StatusOK {
		t.Fatalf("confirming an authenticator = %d", status)
	}

	return setup.Authenticator.Secret, confirmed.RecoveryCodes
}

// A user who turns on an authenticator is asked for it at every sign-in. A
// code works once, a recovery code works once, the answer has to come from
// the browser that signed in, and taking the factor away needs the password.
func TestLiveTwoStepWithAnAuthenticator(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	if f.signIn(owner, challenge).Query().Get("code") == "" {
		t.Fatal("without a second factor the sign-in should go straight through")
	}

	secret, recoveryCodes := addAuthenticator(t, owner)
	if len(recoveryCodes) != 10 {
		t.Fatalf("got %d recovery codes with the first factor, want 10", len(recoveryCodes))
	}

	b := f.heldSignIn(t)

	var offer secondFactorOffer
	if status := b.account(http.MethodGet, "/login/second-factor", nil, &offer); status != http.StatusOK {
		t.Fatalf("GET /login/second-factor = %d", status)
	}
	if want := []string{"totp", "recovery_code"}; !slices.Equal(offer.SecondFactor.Methods, want) {
		t.Errorf("methods = %v, want %v", offer.SecondFactor.Methods, want)
	}

	// Another browser holds no cookie, so it has no sign-in to finish.
	if status, code := answerSecondFactor(f.s.browser(), map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}); status != http.StatusGone || code != "second_factor_expired" {
		t.Errorf("answering from another browser = %d %s, want 410 second_factor_expired", status, code)
	}

	// The code that confirmed the authenticator is spent.
	if status, code := answerSecondFactor(b, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now())-1)}); code != "second_factor_invalid" {
		t.Errorf("a spent code = %d %s, want second_factor_invalid", status, code)
	}

	current := codeAt(t, secret, totp.Step(time.Now()))
	status, to := answerSecondFactor(b, map[string]any{"code": current})
	if back, _ := url.Parse(to); status != http.StatusOK || back.Query().Get("code") == "" {
		t.Fatalf("the right code = %d %s, want the application", status, to)
	}
	if b.account(http.MethodGet, "/me", nil, nil) != http.StatusOK {
		t.Error("the browser is not signed in after its second factor")
	}

	// The same code in another sign-in is refused; a recovery code works,
	// once.
	again := f.heldSignIn(t)
	if _, code := answerSecondFactor(again, map[string]any{"code": current}); code != "second_factor_invalid" {
		t.Errorf("a code used twice = %s, want second_factor_invalid", code)
	}
	if status, _ := answerSecondFactor(again, map[string]any{"code": recoveryCodes[0]}); status != http.StatusOK {
		t.Errorf("a recovery code = %d, want 200", status)
	}
	if _, code := answerSecondFactor(f.heldSignIn(t), map[string]any{"code": recoveryCodes[0]}); code != "second_factor_invalid" {
		t.Errorf("a recovery code used twice = %s, want second_factor_invalid", code)
	}

	var status2 struct {
		TwoStep struct {
			Enabled       bool `json:"enabled"`
			Authenticator *struct {
				ID string `json:"id"`
			} `json:"authenticator"`
			RecoveryCodesLeft int `json:"recovery_codes_left"`
		} `json:"two_step"`
	}
	owner.account(http.MethodGet, "/two-step", nil, &status2)
	if !status2.TwoStep.Enabled || status2.TwoStep.Authenticator == nil || status2.TwoStep.RecoveryCodesLeft != 9 {
		t.Fatalf("two-step = %+v, want an authenticator and nine recovery codes", status2.TwoStep)
	}

	// Taking it away asks for the password.
	path := "/two-step/factors/" + status2.TwoStep.Authenticator.ID
	if status := owner.account(http.MethodDelete, path, map[string]any{"current_password": "not-the-password"}, nil); status == http.StatusNoContent {
		t.Fatal("a factor was removed with the wrong password")
	}
	if status := owner.account(http.MethodDelete, path, map[string]any{"current_password": adaPassword}, nil); status != http.StatusNoContent {
		t.Fatalf("removing the factor = %d", status)
	}

	_, challenge = pkce(t)
	if f.signIn(f.s.browser(), challenge).Query().Get("code") == "" {
		t.Error("with the factor removed, the sign-in still waited for one")
	}
}

// Wrong second factors count towards the account's throttle, as wrong
// passwords do, so guessing across many held sign-ins is bounded too.
func TestLiveTwoStepWrongCodesLockTheAccount(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	_, recoveryCodes := addAuthenticator(t, owner)

	// A right password between wrong codes gives nothing back: three rounds
	// of each spend the five attempts.
	held := f.heldSignIn(t)
	for range 2 {
		answerSecondFactor(held, map[string]any{"code": "000000"})
		held = f.heldSignIn(t)
	}
	answerSecondFactor(held, map[string]any{"code": "000000"})

	// Locked: even a good recovery code is refused, and so is the password.
	if _, code := answerSecondFactor(held, map[string]any{"code": recoveryCodes[0]}); code != "invalid_credentials" {
		t.Errorf("a recovery code on a locked account = %s, want invalid_credentials", code)
	}
	b := f.s.browser()
	_, challenge = pkce(t)
	login := b.visit(f.authorizeURL(challenge, nil))
	if status := b.account(http.MethodPost, "/login", map[string]any{
		"request": login.Query().Get("request"), "email": adaEmail, "password": adaPassword,
	}, nil); status == http.StatusOK {
		t.Error("a locked account could still sign in with its password")
	}
}

// Two-step verification is each user's choice, where the flow offers it.
// Where it does not, nobody can add a factor — but someone who already has
// one is still asked for it.
func TestLiveTwoStepIsOfferedByTheFlow(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)

	var refused struct {
		Code string `json:"code"`
	}
	if status := owner.account(http.MethodPost, "/two-step/authenticator", map[string]any{"current_password": adaPassword}, &refused); status != http.StatusForbidden || refused.Code != "two_step_not_offered" {
		t.Errorf("adding a factor where the flow does not offer it = %d %s", status, refused.Code)
	}

	f.withSteps("identifier", "password", "totp")
	addAuthenticator(t, owner)
	f.withSteps("identifier", "password")

	f.heldSignIn(t)
}

// virtualKey is a passkey authenticator in the test: an ES256 key that
// answers the browser's two ceremonies the way a real one does, with no
// attestation.
type virtualKey struct {
	key   *ecdsa.PrivateKey
	id    []byte
	count uint32
	// user is the handle it was made for, which a passkey sends back when it
	// is the only thing that says who is signing in.
	user string
	// origin is the page the browser says it ran on: the sign-in pages',
	// unless it is the panel's.
	origin string
}

func newVirtualKey(t *testing.T) *virtualKey {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 32)
	_, _ = rand.Read(id)

	return &virtualKey{key: key, id: id, origin: testAccountURL}
}

var b64 = base64.RawURLEncoding

func (k *virtualKey) clientData(t *testing.T, kind string, options map[string]any) []byte {
	t.Helper()

	data, err := json.Marshal(map[string]any{
		"type": kind, "challenge": options["challenge"], "origin": k.origin, "crossOrigin": false,
	})
	if err != nil {
		t.Fatal(err)
	}

	return data
}

func (k *virtualKey) authData(rpID string, flags byte) []byte {
	rp := sha256.Sum256([]byte(rpID))
	data := append(rp[:], flags)

	return binary.BigEndian.AppendUint32(data, k.count)
}

// create answers navigator.credentials.create.
func (k *virtualKey) create(t *testing.T, options map[string]any) json.RawMessage {
	t.Helper()

	rp, _ := options["rp"].(map[string]any)
	rpID, _ := rp["id"].(string)
	user, _ := options["user"].(map[string]any)
	k.user, _ = user["id"].(string)

	public, err := k.key.PublicKey.ECDH()
	if err != nil {
		t.Fatal(err)
	}
	point := public.Bytes() // 0x04 | x | y
	cose, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: point[1:33], -3: point[33:65]})
	if err != nil {
		t.Fatal(err)
	}

	// User present, verified, and carrying the new credential.
	data := k.authData(rpID, 0x01|0x04|0x40)
	data = append(data, make([]byte, 16)...) // AAGUID
	data = binary.BigEndian.AppendUint16(data, uint16(len(k.id)))
	data = append(data, k.id...)
	data = append(data, cose...)

	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "attStmt": map[string]any{}, "authData": data})
	if err != nil {
		t.Fatal(err)
	}

	return k.encode(t, map[string]any{
		"clientDataJSON":    b64.EncodeToString(k.clientData(t, "webauthn.create", options)),
		"attestationObject": b64.EncodeToString(attestation),
		"transports":        []string{"internal"},
	})
}

// get answers navigator.credentials.get, counting up as an authenticator
// does.
func (k *virtualKey) get(t *testing.T, options map[string]any) json.RawMessage {
	t.Helper()

	rpID, _ := options["rpId"].(string)
	k.count++

	data := k.authData(rpID, 0x01|0x04)
	client := k.clientData(t, "webauthn.get", options)
	clientHash := sha256.Sum256(client)
	digest := sha256.Sum256(append(append([]byte{}, data...), clientHash[:]...))

	signature, err := ecdsa.SignASN1(rand.Reader, k.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}

	return k.encode(t, map[string]any{
		"clientDataJSON":    b64.EncodeToString(client),
		"authenticatorData": b64.EncodeToString(data),
		"signature":         b64.EncodeToString(signature),
		"userHandle":        k.user,
	})
}

func (k *virtualKey) encode(t *testing.T, response map[string]any) json.RawMessage {
	t.Helper()

	encoded, err := json.Marshal(map[string]any{
		"id": b64.EncodeToString(k.id), "rawId": b64.EncodeToString(k.id), "type": "public-key",
		"response": response, "clientExtensionResults": map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}

	return encoded
}

// A passkey is a second factor too: added from the account page, asked for
// at sign-in. An answer is for its own sign-in only, and a key whose counter
// goes backwards — a copy — is refused.
func TestLiveTwoStepWithAPasskey(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)

	key := newVirtualKey(t)

	var begun struct {
		Passkey map[string]any `json:"passkey"`
	}
	if status := owner.account(http.MethodPost, "/two-step/passkeys", map[string]any{"current_password": adaPassword}, &begun); status != http.StatusOK {
		t.Fatalf("starting a passkey = %d", status)
	}
	var added struct {
		RecoveryCodes []string `json:"recovery_codes"`
	}
	if status := owner.account(http.MethodPost, "/two-step/passkeys/finish", map[string]any{
		"label": "Test key", "response": key.create(t, begun.Passkey),
	}, &added); status != http.StatusOK || len(added.RecoveryCodes) != 10 {
		t.Fatalf("finishing a passkey = %d, %d recovery codes", status, len(added.RecoveryCodes))
	}

	b := f.heldSignIn(t)
	var offer secondFactorOffer
	b.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if !slices.Contains(offer.SecondFactor.Methods, "passkey") || offer.SecondFactor.Passkey == nil {
		t.Fatalf("the sign-in offers %+v, want a passkey", offer.SecondFactor)
	}

	answer := key.get(t, offer.SecondFactor.Passkey)
	if status, to := answerSecondFactor(b, map[string]any{"passkey": answer}); status != http.StatusOK || to == "" {
		t.Fatalf("the passkey = %d %s, want the application", status, to)
	}

	// The same answer cannot finish another sign-in: its challenge is not
	// that sign-in's.
	other := f.heldSignIn(t)
	other.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if _, code := answerSecondFactor(other, map[string]any{"passkey": answer}); code != "second_factor_invalid" {
		t.Errorf("an answer replayed into another sign-in = %s, want second_factor_invalid", code)
	}

	// A challenge is spent on its first answer, even a wrong one: the right
	// answer to it afterwards is refused, and a fresh challenge is needed.
	if _, code := answerSecondFactor(other, map[string]any{"passkey": key.get(t, offer.SecondFactor.Passkey)}); code != "second_factor_invalid" {
		t.Errorf("the right answer to a challenge already answered = %s, want second_factor_invalid", code)
	}

	// A copy of the key, whose counter is behind the original's, is refused.
	other.account(http.MethodGet, "/login/second-factor", nil, &offer)
	saved := key.count
	key.count = 0
	if _, code := answerSecondFactor(other, map[string]any{"passkey": key.get(t, offer.SecondFactor.Passkey)}); code != "second_factor_invalid" {
		t.Errorf("a key whose counter went backwards = %s, want second_factor_invalid", code)
	}

	// The real key, on a fresh challenge, still signs in.
	key.count = saved
	other.account(http.MethodGet, "/login/second-factor", nil, &offer)
	if status, to := answerSecondFactor(other, map[string]any{"passkey": key.get(t, offer.SecondFactor.Passkey)}); status != http.StatusOK || to == "" {
		t.Errorf("the key on a fresh challenge = %d %s, want the application", status, to)
	}
}

// An administrator can turn a user's two-step verification off, for someone
// who lost every factor and recovery code; the panel can tell who has it.
func TestLiveAdministratorResetsTwoStep(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	addAuthenticator(t, owner)

	enabled := func() bool {
		var out struct {
			User struct {
				TwoStepEnabled bool `json:"two_step_enabled"`
			} `json:"user"`
		}
		f.super.must(http.StatusOK, http.MethodGet, "/users/"+f.userID, nil, &out)
		return out.User.TwoStepEnabled
	}

	if !enabled() {
		t.Fatal("the panel does not see the user's two-step verification")
	}

	// The list says it too: the user dialog opens from a row.
	var page struct {
		Users []struct {
			Email          string `json:"email"`
			TwoStepEnabled bool   `json:"two_step_enabled"`
		} `json:"users"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/users?search=ada", nil, &page)
	if len(page.Users) != 1 || !page.Users[0].TwoStepEnabled {
		t.Errorf("the user list shows %+v, want ada with two-step verification", page.Users)
	}

	f.super.must(http.StatusNoContent, http.MethodDelete, "/users/"+f.userID+"/two-step", nil, nil)

	if enabled() {
		t.Error("the user still has two-step verification after the reset")
	}
	_, challenge = pkce(t)
	if f.signIn(f.s.browser(), challenge).Query().Get("code") == "" {
		t.Error("after the reset, the sign-in still waited for a second factor")
	}
}

// Another account is a first factor like a password: someone with two-step
// verification who signs in through a social provider lands on the
// second-factor page, with no session, and is signed in only once they give
// it.
func TestLiveTwoStepAfterASocialSignIn(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	secret, _ := addAuthenticator(t, owner)

	provider := newFakeProvider(t, map[string]any{
		"sub": "ada-at-the-provider", "email": adaEmail, "email_verified": true,
	})
	b, start := provider.register(t, f.s, f.super, nil)

	landed := through(t, b, start)
	if !strings.HasPrefix(landed.String(), testAccountURL+"/second-factor") {
		t.Fatalf("the provider's callback led to %s, want the second-factor page", landed)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Fatal("the callback signed the browser in before its second factor")
	}

	var offer secondFactorOffer
	if status := b.account(http.MethodGet, "/login/second-factor", nil, &offer); status != http.StatusOK {
		t.Fatalf("GET /login/second-factor after the callback = %d", status)
	}
	if !slices.Contains(offer.SecondFactor.Methods, "totp") {
		t.Errorf("methods = %v, want the authenticator", offer.SecondFactor.Methods)
	}

	if status, _ := answerSecondFactor(b, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}); status != http.StatusOK {
		t.Fatalf("the right code after a social sign-in = %d", status)
	}
	if status := b.account(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Error("the browser is not signed in after its second factor")
	}
}

// addPasskey gives whoever b is signed in as a passkey, and returns it.
func addPasskey(t *testing.T, b *browser) *virtualKey {
	t.Helper()

	key := newVirtualKey(t)
	var begun struct {
		Passkey map[string]any `json:"passkey"`
	}
	if status := b.account(http.MethodPost, "/two-step/passkeys", map[string]any{"current_password": adaPassword}, &begun); status != http.StatusOK {
		t.Fatalf("starting a passkey = %d", status)
	}
	if status := b.account(http.MethodPost, "/two-step/passkeys/finish", map[string]any{
		"label": "Test key", "response": key.create(t, begun.Passkey),
	}, nil); status != http.StatusOK {
		t.Fatalf("finishing a passkey = %d", status)
	}

	return key
}

// A passkey signs in on its own, with nothing typed and no second factor
// after it: possession and the device's own check are both factors. Each
// answer is good once, only a passkey the account has works, and only where
// the flow offers passkeys at all.
func TestLiveSignInWithAPasskeyAlone(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	key := addPasskey(t, owner)

	start := func(b *browser) (string, map[string]any) {
		t.Helper()

		_, challenge := pkce(t)
		login := b.visit(f.authorizeURL(challenge, nil))
		var offer struct {
			Handle  string         `json:"handle"`
			Passkey map[string]any `json:"passkey"`
		}
		if status := b.account(http.MethodPost, "/login/passkey", map[string]any{
			"request": login.Query().Get("request"),
		}, &offer); status != http.StatusOK {
			t.Fatalf("starting a passkey sign-in = %d", status)
		}
		return offer.Handle, offer.Passkey
	}
	finish := func(b *browser, handle string, answer json.RawMessage) (int, string) {
		var out struct {
			RedirectTo string `json:"redirect_to"`
			Code       string `json:"code"`
		}
		status := b.account(http.MethodPost, "/login/passkey/finish", map[string]any{
			"handle": handle, "response": answer,
		}, &out)
		if status == http.StatusOK {
			return status, out.RedirectTo
		}
		return status, out.Code
	}

	b := f.s.browser()
	handle, options := start(b)
	answer := key.get(t, options)
	status, to := finish(b, handle, answer)
	if back, _ := url.Parse(to); status != http.StatusOK || back.Query().Get("code") == "" {
		t.Fatalf("a passkey sign-in = %d %s, want the application", status, to)
	}
	if b.account(http.MethodGet, "/me", nil, nil) != http.StatusOK {
		t.Error("the browser is not signed in after its passkey")
	}

	if status, code := finish(f.s.browser(), handle, answer); status != http.StatusGone || code != "request_expired" {
		t.Errorf("the same answer again = %d %s, want 410 request_expired", status, code)
	}

	stranger := newVirtualKey(t)
	stranger.user = key.user
	handle, options = start(f.s.browser())
	if _, code := finish(f.s.browser(), handle, stranger.get(t, options)); code != "passkey_sign_in_failed" {
		t.Errorf("a passkey the account does not have = %s, want passkey_sign_in_failed", code)
	}

	f.withSteps("identifier", "password")
	var refused struct {
		Code string `json:"code"`
	}
	if status := f.s.browser().account(http.MethodPost, "/login/passkey", map[string]any{}, &refused); status != http.StatusForbidden || refused.Code != "two_step_not_offered" {
		t.Errorf("a passkey sign-in where the flow offers none = %d %s", status, refused.Code)
	}
}

// Adding a way to sign in asks for the password, as taking one away does: a
// session left open, or stolen, cannot add a passkey of its own and keep a
// way in after the password changes.
func TestLiveAddingAFactorAsksForThePassword(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)

	for _, path := range []string{"/two-step/authenticator", "/two-step/passkeys"} {
		for _, password := range []string{"", "not-the-password"} {
			if status := owner.account(http.MethodPost, path, map[string]any{"current_password": password}, nil); status == http.StatusOK {
				t.Errorf("POST %s with password %q was let through", path, password)
			}
		}
	}
}
