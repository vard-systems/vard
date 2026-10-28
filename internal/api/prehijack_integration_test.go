package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"vard/internal/mail"
	"vard/internal/model"
)

// Pre-hijacking: someone registers an address before its owner does, gives
// the account a way back in, and waits. When the owner later proves the
// address — with a reset link or the verification link — everything the
// account held before that moment goes, so the stranger's way back in goes
// with it.

const victimEmail = "victim@example.com"

// preRegister makes the account for victimEmail the way a stranger would: by
// registering it at an application, unverified, with adaPassword (so the
// two-step helpers, which confirm with it, work for them too). The flow lets
// an unverified address in, as an existing installation's may.
func (f *oauthFixture) preRegister(t *testing.T) *browser {
	t.Helper()

	f.super.must(http.StatusOK, http.MethodPatch, "/applications/"+f.appID, map[string]any{
		"name": "Shop", "type": "web",
		"grant_types":        []string{"authorization_code", "refresh_token"},
		"redirect_uris":      []string{shopRedirect},
		"scopes":             []string{"openid", "profile", "email", "offline_access", "roles"},
		"allow_registration": true,
	}, nil)
	f.super.must(http.StatusOK, http.MethodPatch, "/login-flows/"+defaultFlow(t, f.super), map[string]any{
		"steps":                    []string{"identifier", "password", "social", "totp"},
		"require_verified_email":   false,
		"verify_email_on_register": true,
	}, nil)

	stranger := f.s.browser()
	_, challenge := pkce(t)
	login := stranger.visit(f.authorizeURL(challenge, nil))
	if status := stranger.account(http.MethodPost, "/register", map[string]any{
		"request": login.Query().Get("request"), "email": victimEmail, "password": adaPassword,
	}, nil); status != http.StatusOK {
		t.Fatalf("the stranger registering the address = %d", status)
	}

	return stranger
}

// held counts what the account at victimEmail still holds.
func (f *oauthFixture) held(t *testing.T, table string) int64 {
	t.Helper()

	var count int64
	err := f.s.db.Table(table).
		Joins("JOIN users ON users.id = "+table+".user_id").
		Where("users.email = ?", victimEmail).
		Count(&count).Error
	if err != nil {
		t.Fatal(err)
	}

	return count
}

func TestLiveProvingAnAddressEndsWhatAStrangerSetUpBeforeIt(t *testing.T) {
	ownerResets := func(t *testing.T, f *oauthFixture) {
		f.s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": victimEmail}, nil)
		token := tokenIn(t, f.s.mail.waitFor(t, victimEmail, "/reset-password").Body, "/reset-password")
		if status := f.s.browser().account(http.MethodPost, "/reset-password", map[string]string{
			"token": token, "password": "owner-chose-this-1",
		}, nil); status != http.StatusOK {
			t.Fatalf("the owner's reset = %d", status)
		}
	}
	ownerVerifies := func(t *testing.T, f *oauthFixture) {
		token := tokenIn(t, f.s.mail.waitFor(t, victimEmail, "/verify-email").Body, "/verify-email")
		if status := f.s.browser().account(http.MethodPost, "/verify-email", map[string]string{"token": token}, nil); status != http.StatusOK {
			t.Fatalf("the owner's verification = %d", status)
		}
	}

	cases := []struct {
		name  string
		setUp func(t *testing.T, stranger *browser) *virtualKey
		prove func(t *testing.T, f *oauthFixture)
	}{
		{
			name: "an authenticator, then the owner resets the password",
			setUp: func(t *testing.T, stranger *browser) *virtualKey {
				addAuthenticator(t, stranger)
				return nil
			},
			prove: ownerResets,
		},
		{
			name:  "a passkey, then the owner verifies the address",
			setUp: addPasskey,
			prove: ownerVerifies,
		},
		{
			name:  "a passkey, then the owner resets the password",
			setUp: addPasskey,
			prove: ownerResets,
		},
		{
			name: "an authenticator, then the owner verifies the address",
			setUp: func(t *testing.T, stranger *browser) *virtualKey {
				addAuthenticator(t, stranger)
				return nil
			},
			prove: ownerVerifies,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newOAuthFixture(t)
			stranger := f.preRegister(t)
			key := c.setUp(t, stranger)

			// What the stranger allowed the application stays allowed for the
			// owner unless it goes too.
			if err := f.s.db.Exec(`INSERT INTO consents (id, created_at, updated_at, user_id, application_id, scopes)
				SELECT gen_random_uuid(), now(), now(), id, ?, '["openid"]' FROM users WHERE email = ?`, f.appID, victimEmail).Error; err != nil {
				t.Fatal(err)
			}

			c.prove(t, f)

			for _, table := range []string{"user_factors", "user_recovery_codes", "consents", "pending_sign_ins"} {
				if left := f.held(t, table); left != 0 {
					t.Errorf("%s still holds %d rows set up before the address was proved", table, left)
				}
			}
			if status := stranger.account(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
				t.Errorf("the stranger's session = %d after the owner proved the address, want 401", status)
			}

			if key != nil {
				b := f.s.browser()
				_, challenge := pkce(t)
				login := b.visit(f.authorizeURL(challenge, nil))
				var offer struct {
					Handle  string         `json:"handle"`
					Passkey map[string]any `json:"passkey"`
				}
				b.account(http.MethodPost, "/login/passkey", map[string]any{"request": login.Query().Get("request")}, &offer)
				var out struct {
					RedirectTo string `json:"redirect_to"`
				}
				if status := b.account(http.MethodPost, "/login/passkey/finish", map[string]any{
					"handle": offer.Handle, "response": key.get(t, offer.Passkey),
				}, &out); status == http.StatusOK {
					t.Errorf("the stranger's passkey still signs in after the owner proved the address: %s", out.RedirectTo)
				}
			}

			var audit int64
			if err := f.s.db.Model(&model.AuditLog{}).Where("action = ?", "user.address_proved").Count(&audit).Error; err != nil {
				t.Fatal(err)
			}
			if audit != 1 {
				t.Errorf("%d user.address_proved entries, want 1", audit)
			}
		})
	}
}

// An account already verified keeps its factors through a reset: only the
// first proof of an address clears what came before it.
func TestLiveAResetOfAVerifiedAccountKeepsItsFactors(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "totp")

	owner := f.s.browser()
	_, challenge := pkce(t)
	f.signIn(owner, challenge)
	addAuthenticator(t, owner)

	f.s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": adaEmail}, nil)
	token := tokenIn(t, f.s.mail.waitFor(t, adaEmail, "/reset-password").Body, "/reset-password")
	if status := f.s.browser().account(http.MethodPost, "/reset-password", map[string]string{
		"token": token, "password": "ada-new-password-1",
	}, nil); status != http.StatusOK {
		t.Fatalf("reset = %d", status)
	}

	var left int64
	if err := f.s.db.Table("user_factors").Where("user_id = ?", f.userID).Count(&left).Error; err != nil {
		t.Fatal(err)
	}
	if left != 1 {
		t.Errorf("a verified account's reset left %d factors, want its 1", left)
	}
}

// A provider that has not verified an address may not make an account for
// it: whoever controls that provider account would hold it before its owner.
func TestLiveSocialSignInRefusesToMakeAnAccountForAnUnverifiedAddress(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	provider := newFakeProvider(t, map[string]any{
		"sub": "stranger-1", "email": victimEmail, "email_verified": false,
	})
	b, start := provider.register(t, s, super, nil)

	landed := through(t, b, start)
	if !strings.HasPrefix(landed.String(), testAccountURL+"/error") || landed.Query().Get("reason") != "social_email_unverified" {
		t.Fatalf("an unverified address landed at %s, want the error page with social_email_unverified", landed)
	}

	var users struct {
		Users []json.RawMessage `json:"users"`
	}
	super.must(http.StatusOK, http.MethodGet, "/users?search="+url.QueryEscape(victimEmail), nil, &users)
	if len(users.Users) != 0 {
		t.Errorf("%d accounts were made for an address the provider did not verify", len(users.Users))
	}
}

// An identity linked before the owner proved the address — on an
// installation from before that was refused — signs nobody in afterwards.
func TestLiveAStrangersSocialLinkDiesWithTheOwnersReset(t *testing.T) {
	f := newOAuthFixture(t)
	f.preRegister(t)

	provider := newFakeProvider(t, map[string]any{
		"sub": "stranger-1", "email": victimEmail, "email_verified": false,
	})
	b, start := provider.register(t, f.s, f.super, nil)

	var providerID, userID string
	if err := f.s.db.Table("social_providers").Select("id").Where("slug = ?", "fake").Scan(&providerID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.Table("users").Select("id").Where("email = ?", victimEmail).Scan(&userID).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.s.db.Exec(`INSERT INTO social_identities (id, created_at, updated_at, user_id, provider_id, subject, email)
		VALUES (gen_random_uuid(), now(), now(), ?, ?, 'stranger-1', ?)`, userID, providerID, victimEmail).Error; err != nil {
		t.Fatal(err)
	}

	if landed := through(t, b, start); strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the planted link did not sign in before the reset, so it proves nothing: %s", landed)
	}

	f.s.browser().account(http.MethodPost, "/forgot-password", map[string]string{"email": victimEmail}, nil)
	token := tokenIn(t, f.s.mail.waitFor(t, victimEmail, "/reset-password").Body, "/reset-password")
	if status := f.s.browser().account(http.MethodPost, "/reset-password", map[string]string{
		"token": token, "password": "owner-chose-this-1",
	}, nil); status != http.StatusOK {
		t.Fatalf("the owner's reset = %d", status)
	}

	again := f.s.browser()
	if landed := through(t, again, start); !strings.HasPrefix(landed.String(), testAccountURL+"/error") {
		t.Fatalf("the stranger's social sign-in after the owner's reset landed at %s, want refused", landed)
	}
	if status := again.account(http.MethodGet, "/me", nil, nil); status == http.StatusOK {
		t.Error("the stranger is signed in to the owner's account")
	}
}

// waitFor is wait for the first message to `to` carrying a link to page.
func (m *mailbox) waitFor(t *testing.T, to, page string) mail.Message {
	t.Helper()

	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		m.mu.Lock()
		for _, msg := range m.messages {
			if strings.EqualFold(msg.To, to) && strings.Contains(msg.Body, testAccountURL+page) {
				m.mu.Unlock()
				return msg
			}
		}
		m.mu.Unlock()
	}

	t.Fatalf("no email to %s with a %s link", to, page)
	return mail.Message{}
}
