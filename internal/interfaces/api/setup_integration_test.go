package api

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"vard/internal/application/auth"
)

// setUp posts the first administrator's form with a token, and returns the
// status and the problem's code.
func (s *liveServer) setUp(email, token string) (int, string) {
	var out struct {
		Code string `json:"code"`
	}
	status := s.client().do(http.MethodPost, "/setup", map[string]string{
		"email": email, "password": "first-password-1", "first_name": "First", "setup_token": token,
	}, &out)

	return status, out.Code
}

// Whoever reaches a fresh panel first cannot make themselves its
// administrator: that takes the token the server logged, once.
func TestLiveSettingUpTakesTheLoggedToken(t *testing.T) {
	cases := []struct {
		name  string
		token func(s *liveServer) string
	}{
		{"no token", func(*liveServer) string { return "" }},
		{"a token the server never issued", func(*liveServer) string { return randomHex(t, 32) }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newLiveServer(t)

			if status, code := s.setUp("intruder@example.com", c.token(s)); status != http.StatusForbidden || code != "setup_token_invalid" {
				t.Errorf("setting up with %s = %d %s, want 403 setup_token_invalid", c.name, status, code)
			}

			var admins int64
			if err := s.db.Table("admins").Count(&admins).Error; err != nil {
				t.Fatal(err)
			}
			if admins != 0 {
				t.Errorf("%d administrators exist after a refused setup", admins)
			}
		})
	}
}

// The token works once; a second token issued meanwhile (a second replica's,
// say) is no use once someone is an administrator, and no new one is issued.
func TestLiveASetupTokenWorksOnce(t *testing.T) {
	s := newLiveServer(t)

	second, err := auth.IssueSetupToken(context.Background(), s.store, time.Now())
	if err != nil || second == "" {
		t.Fatalf("a second token while nobody is an administrator = %q %v", second, err)
	}

	if status, code := s.setUp("first@example.com", s.setupToken); status != http.StatusCreated {
		t.Fatalf("setting up with the logged token = %d %s", status, code)
	}
	if status, code := s.setUp("again@example.com", s.setupToken); status != http.StatusConflict || code != "setup_done" {
		t.Errorf("the same token again = %d %s, want 409 setup_done", status, code)
	}
	if status, code := s.setUp("other@example.com", second); status != http.StatusConflict || code != "setup_done" {
		t.Errorf("another replica's token after setup = %d %s, want 409 setup_done", status, code)
	}

	if after, err := auth.IssueSetupToken(context.Background(), s.store, time.Now()); err != nil || after != "" {
		t.Errorf("a server started after setup issued %q %v, want none", after, err)
	}
}

// A token past its lifetime is refused like a wrong one.
func TestLiveAnExpiredSetupTokenIsRefused(t *testing.T) {
	s := newLiveServer(t)

	if err := s.db.Exec("UPDATE setup_tokens SET expires_at = now() - interval '1 second'").Error; err != nil {
		t.Fatal(err)
	}
	if status, code := s.setUp("late@example.com", s.setupToken); status != http.StatusForbidden || code != "setup_token_invalid" {
		t.Errorf("an expired token = %d %s, want 403 setup_token_invalid", status, code)
	}
}

// Two setups racing with valid tokens make one administrator.
func TestLiveRacingSetupsMakeOneAdministrator(t *testing.T) {
	s := newLiveServer(t)

	tokens := []string{s.setupToken}
	for range 4 {
		token, err := auth.IssueSetupToken(context.Background(), s.store, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, token, token)
	}

	var wg sync.WaitGroup
	statuses := make([]int, len(tokens))
	for i, token := range tokens {
		wg.Go(func() {
			statuses[i], _ = s.setUp("racer"+string(rune('a'+i))+"@example.com", token)
		})
	}
	wg.Wait()

	created := 0
	for _, status := range statuses {
		if status == http.StatusCreated {
			created++
		}
	}
	var admins int64
	if err := s.db.Table("admins").Count(&admins).Error; err != nil {
		t.Fatal(err)
	}
	if created != 1 || admins != 1 {
		t.Errorf("%d setups succeeded and %d administrators exist, want one of each (statuses %v)", created, admins, statuses)
	}
}
