package api

import (
	"net/http"
	"sync"
	"testing"
)

// setUp posts the first administrator's form, and returns the status and the
// problem's code.
func (s *liveServer) setUp(email string) (int, string) {
	var out struct {
		Code string `json:"code"`
	}
	status := s.client().do(http.MethodPost, "/setup", map[string]string{
		"email": email, "password": "first-password-1", "first_name": "First",
	}, &out)

	return status, out.Code
}

// A fresh panel is set up once: the first form makes its administrator, and
// every one after is refused.
func TestLiveThePanelIsSetUpOnce(t *testing.T) {
	s := newLiveServer(t)

	if status, code := s.setUp("first@example.com"); status != http.StatusCreated {
		t.Fatalf("the first setup = %d %s, want 201", status, code)
	}
	if status, code := s.setUp("again@example.com"); status != http.StatusConflict || code != "setup_done" {
		t.Errorf("a second setup = %d %s, want 409 setup_done", status, code)
	}
}

// Setups sent at once still make one administrator: the store counts them
// under a lock.
func TestLiveRacingSetupsMakeOneAdministrator(t *testing.T) {
	s := newLiveServer(t)

	var wg sync.WaitGroup
	statuses := make([]int, 8)
	for i := range statuses {
		wg.Go(func() {
			statuses[i], _ = s.setUp("racer" + string(rune('a'+i)) + "@example.com")
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
