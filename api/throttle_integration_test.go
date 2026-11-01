package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"vard/internal/config"
	"vard/internal/domain/model"
	"vard/internal/domain/totp"
)

// Users' sign-ins are throttled, not locked: past a few failures each attempt
// waits longer, counted against the address typed whoever types it, and the
// wait ends on its own.

// throttledFixture is an OAuth fixture whose server believes the caller's
// address from X-Forwarded-For, as it does from the id app's server, so one
// test can sign in from many addresses.
func throttledFixture(t *testing.T) *oauthFixture {
	return newOAuthFixtureWith(t, func(cfg *config.Config) {
		cfg.TrustedProxies = []string{"127.0.0.1/32", "::1/128"}
	})
}

// signInFrom posts a sign-in from `address` in this browser, and answers the
// status and the problem's code.
func (b *browser) signInFrom(address, email, password string) (int, string) {
	b.t.Helper()

	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequest(http.MethodPost, b.s.root+"/api/v1/account/login", bytes.NewReader(body))
	if err != nil {
		b.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", address)

	res, err := b.http.Do(req)
	if err != nil {
		b.t.Fatal(err)
	}
	defer res.Body.Close()

	var out struct {
		Code string `json:"code"`
	}
	_ = json.NewDecoder(res.Body).Decode(&out)

	return res.StatusCode, out.Code
}

// waitOver ends every wait, as time passing would.
func (f *oauthFixture) waitOver(t *testing.T) {
	t.Helper()
	if err := f.s.db.Exec("UPDATE sign_in_throttles SET blocked_until = now() - interval '1 second'").Error; err != nil {
		t.Fatal(err)
	}
}

// A guesser spreading one account's attempts over many addresses is held to
// the account's count all the same, and the wait ends on its own.
func TestLiveASprayFromManyAddressesIsThrottledPerAccount(t *testing.T) {
	f := throttledFixture(t)

	for i := range 6 {
		if status, _ := f.s.browser().signInFrom(fmt.Sprintf("198.51.100.%d", i+1), adaEmail, "wrong-password"); status != http.StatusUnauthorized {
			t.Fatalf("guess %d = %d, want 401", i+1, status)
		}
	}

	if status, code := f.s.browser().signInFrom("198.51.100.99", adaEmail, adaPassword); status != http.StatusUnauthorized || code != "invalid_credentials" {
		t.Errorf("the right password from a new address while throttled = %d %s, want 401 invalid_credentials", status, code)
	}

	f.waitOver(t)
	if status, code := f.s.browser().signInFrom("198.51.100.100", adaEmail, adaPassword); status != http.StatusOK {
		t.Errorf("the right password once the wait is over = %d %s, want 200", status, code)
	}
}

// Each wait past the free failures is longer than the last.
func TestLiveEachWaitIsLongerThanTheLast(t *testing.T) {
	f := throttledFixture(t)

	wait := func() time.Duration {
		t.Helper()
		var seconds float64
		err := f.s.db.Raw(`SELECT COALESCE(MAX(EXTRACT(EPOCH FROM blocked_until - now())), 0) FROM sign_in_throttles WHERE key LIKE 'email:%'`).Scan(&seconds).Error
		if err != nil {
			t.Fatal(err)
		}
		return time.Duration(seconds * float64(time.Second))
	}

	for range 6 {
		f.s.browser().signInFrom("198.51.100.1", adaEmail, "wrong-password")
	}
	first := wait()

	f.waitOver(t)
	f.s.browser().signInFrom("198.51.100.1", adaEmail, "wrong-password")
	second := wait()

	if first < 20*time.Second || second < first+20*time.Second {
		t.Errorf("waits = %v then %v, want about 30s then a minute", first, second)
	}
}

// An address nobody has is answered and throttled exactly as one somebody
// has, in wording and in time, so neither says which accounts exist.
func TestLiveAnUnknownAddressIsThrottledLikeAKnownOne(t *testing.T) {
	f := throttledFixture(t)

	answers := map[string][]string{}
	took := map[string]time.Duration{}
	for _, email := range []string{adaEmail, "nobody@example.com"} {
		for i := range 7 {
			start := time.Now()
			status, code := f.s.browser().signInFrom(fmt.Sprintf("198.51.100.%d", i+1), email, "wrong-password")
			took[email] += time.Since(start)
			answers[email] = append(answers[email], fmt.Sprintf("%d %s", status, code))
		}
	}

	if fmt.Sprint(answers[adaEmail]) != fmt.Sprint(answers["nobody@example.com"]) {
		t.Errorf("answers differ:\n  an account: %v\n  nobody:     %v", answers[adaEmail], answers["nobody@example.com"])
	}

	known, unknown := took[adaEmail], took["nobody@example.com"]
	if ratio := float64(known) / float64(unknown); ratio < 0.5 || ratio > 2 {
		t.Errorf("seven attempts took %v for an account and %v for nobody, want about the same", known, unknown)
	}
}

// The owner who resets the password through the emailed link is not kept
// waiting by a stranger's guesses at their address.
func TestLiveAResetEndsTheWait(t *testing.T) {
	f := throttledFixture(t)

	for i := range 6 {
		f.s.browser().signInFrom(fmt.Sprintf("198.51.100.%d", i+1), adaEmail, "wrong-password")
	}

	f.s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": adaEmail}, nil)
	token := tokenIn(t, f.s.mail.waitFor(t, adaEmail, "/reset-password").Body, "/reset-password")
	if status := f.s.browser().account(http.MethodPost, "/reset-password", map[string]string{
		"token": token, "password": "ada-new-password-1",
	}, nil); status != http.StatusOK {
		t.Fatalf("reset = %d", status)
	}

	if status, code := f.s.browser().signInFrom("198.51.100.50", adaEmail, "ada-new-password-1"); status != http.StatusOK {
		t.Errorf("the new password straight after the reset = %d %s, want 200", status, code)
	}
}

// Asking for a reset link takes as long for an address nobody has as for one
// somebody has: everything a known address costs happens after the answer.
func TestLiveForgotPasswordTakesAsLongForAnyAddress(t *testing.T) {
	f := newOAuthFixture(t)

	ask := func(email string) time.Duration {
		start := time.Now()
		if status := f.s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": email}, nil); status != http.StatusAccepted {
			t.Fatalf("forgot-password = %d", status)
		}
		return time.Since(start)
	}

	// Warm both paths, then take the fastest of several, which is the
	// work itself rather than the machine's noise.
	ask(adaEmail)
	ask("nobody@example.com")
	fastest := func(email string) time.Duration {
		best := time.Hour
		for range 15 {
			f.s.db.Exec("DELETE FROM password_resets")
			best = min(best, ask(email))
		}
		return best
	}

	known, unknown := fastest(adaEmail), fastest("nobody@example.com")
	if ratio := float64(known) / float64(unknown); ratio > 1.5 {
		t.Errorf("an account's address took %v and nobody's %v, want about the same", known, unknown)
	}
}

// networkFailures is the count against `address`'s network, 0 without one.
func (f *oauthFixture) networkFailures(t *testing.T, address string) int {
	t.Helper()
	var failures int
	err := f.s.db.Raw("SELECT COALESCE(MAX(failures), 0) FROM sign_in_throttles WHERE key = ?", "network:"+address).
		Scan(&failures).Error
	if err != nil {
		t.Fatal(err)
	}
	return failures
}

// A network at the edge of its free attempts, as an office's typos leave it,
// is not made to wait by somebody signing in correctly from it.
func TestLiveARightPasswordDoesNotMakeItsNetworkWait(t *testing.T) {
	f := throttledFixture(t)
	const office = "203.0.113.7"

	err := f.s.db.Exec(`INSERT INTO sign_in_throttles (id, created_at, updated_at, key, failures, expires_at)
		VALUES (gen_random_uuid(), now(), now(), ?, ?, now() + interval '1 hour')`,
		"network:"+office, model.ThrottleFreeNetwork-1).Error
	if err != nil {
		t.Fatal(err)
	}

	for i := range 2 {
		if status, code := f.s.browser().signInFrom(office, adaEmail, adaPassword); status != http.StatusOK {
			t.Fatalf("right password %d from the office = %d %s, want 200", i+1, status, code)
		}
	}
	if got := f.networkFailures(t, office); got != model.ThrottleFreeNetwork-1 {
		t.Errorf("the office's count = %d after two right passwords, want it left at %d", got, model.ThrottleFreeNetwork-1)
	}
}

// A sign-in with a second factor takes nothing from its network: the right
// password and the right code each give back the attempt they took.
func TestLiveASignInWithASecondFactorLeavesItsNetworkAsItWas(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "social", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	secret, _ := addAuthenticator(t, owner)

	before := f.networkFailures(t, "127.0.0.1")
	held := f.heldSignIn(t)
	if status, code := answerSecondFactor(held, map[string]any{"code": codeAt(t, secret, totp.Step(time.Now()))}); status != http.StatusOK {
		t.Fatalf("the right code = %d %s, want 200", status, code)
	}
	if after := f.networkFailures(t, "127.0.0.1"); after != before {
		t.Errorf("the network's count went from %d to %d over a right sign-in, want it unchanged", before, after)
	}
}
