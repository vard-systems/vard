package api

import (
	"net/http"
	"net/url"
	"testing"
)

// Allowing a device is a sign-in to its application, so its flow's
// Agreements step holds there as it does at authorize: the device page says
// the terms are to be accepted, a plain "allow" is refused and the device
// keeps waiting, and allowing it with the terms accepts them.
func TestLiveAllowingADeviceAsksForTheTerms(t *testing.T) {
	f := newOAuthFixture(t)
	f.withSteps("identifier", "password", "terms")
	f.super.must(http.StatusOK, http.MethodPatch, "/organization", map[string]any{"terms_url": "https://example.com/terms"}, nil)

	tvID, _, _ := f.register(map[string]any{
		"name": "TV", "type": "native",
		"grant_types": []string{"urn:ietf:params:oauth:grant-type:device_code"},
		"scopes":      []string{"openid", "email"},
	})
	start := func() (userCode string, poll func() tokenResult) {
		started := f.token("/oauth2/device_authorization", tvID, "", url.Values{"scope": {"openid email"}})
		if started.status != http.StatusOK {
			t.Fatalf("starting a device = %d %v", started.status, started.body)
		}
		deviceCode := started.str("device_code")
		return started.str("user_code"), func() tokenResult {
			return f.token("/oauth2/token", tvID, "", url.Values{
				"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
				"device_code": {deviceCode},
			})
		}
	}
	shown := func(b *browser, userCode string) bool {
		var page struct {
			Device struct {
				Agreements bool `json:"agreements"`
			} `json:"device"`
		}
		if status := b.account(http.MethodGet, "/device?user_code="+url.QueryEscape(userCode), nil, &page); status != http.StatusOK {
			t.Fatalf("GET /device = %d", status)
		}
		return page.Device.Agreements
	}

	b := f.s.browser()
	if status := b.signInAs(adaEmail, adaPassword, "", false, nil); status != http.StatusOK {
		t.Fatalf("signing in = %d", status)
	}

	userCode, poll := start()
	if !shown(b, userCode) {
		t.Error("the device page does not say the terms are to be accepted")
	}

	var problem problemBody
	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": userCode, "yes": true}, fresh(&problem)); status != http.StatusForbidden || problem.Code != "agreements_required" {
		t.Fatalf("allowing without the terms = %d %s, want agreements_required", status, problem.Code)
	}
	if got := poll().str("error"); got != "authorization_pending" {
		t.Errorf("after the refusal the device is told %q, want authorization_pending", got)
	}

	// Saying no needs nothing accepted.
	deniedCode, deniedPoll := start()
	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": deniedCode, "yes": false}, nil); status != http.StatusNoContent {
		t.Errorf("denying a device = %d, want 204", status)
	}
	if got := deniedPoll().str("error"); got != "access_denied" {
		t.Errorf("a denied device is told %q, want access_denied", got)
	}

	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": userCode, "yes": true, "accept_terms": true}, nil); status != http.StatusNoContent {
		t.Fatalf("allowing with the terms = %d", status)
	}
	if tokens := poll(); tokens.status != http.StatusOK {
		t.Fatalf("the device's tokens = %d %v", tokens.status, tokens.body)
	}

	var accepted int64
	if err := f.s.db.Table("users").Where("id = ? AND agreements_accepted_at IS NOT NULL", f.userID).Count(&accepted).Error; err != nil {
		t.Fatal(err)
	}
	if accepted != 1 {
		t.Error("allowing the device with the terms did not record them accepted")
	}

	// Accepted once, the next device asks nothing more, as authorize does.
	next, nextPoll := start()
	if shown(b, next) {
		t.Error("the terms are asked for again after they were accepted")
	}
	if status := b.account(http.MethodPost, "/device", map[string]any{"user_code": next, "yes": true}, nil); status != http.StatusNoContent {
		t.Fatalf("allowing a device after the terms = %d", status)
	}
	if tokens := nextPoll(); tokens.status != http.StatusOK {
		t.Errorf("the next device's tokens = %d %v", tokens.status, tokens.body)
	}
}
