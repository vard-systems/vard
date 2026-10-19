package api

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

// raw asks for a path and returns its body as text, for an answer that is
// not JSON — the export's CSV.
func (c *client) raw(method, path string) (string, int) {
	c.s.t.Helper()

	req, err := http.NewRequest(method, c.s.url+path, nil)
	if err != nil {
		c.s.t.Fatal(err)
	}

	res, err := c.http.Do(req)
	if err != nil {
		c.s.t.Fatal(err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		c.s.t.Fatal(err)
	}

	return string(body), res.StatusCode
}

type ownSessions struct {
	Sessions []struct {
		ID      string `json:"id"`
		Active  bool   `json:"active"`
		Current bool   `json:"current"`
	} `json:"sessions"`
}

// An administrator sees where they are signed in, which of those is the
// browser asking, and can end the others — one at a time or all at once —
// but not the one they are using, and not anyone else's.
func TestLiveOwnSessions(t *testing.T) {
	s := newLiveServer(t)
	here := s.superAdmin()

	elsewhere := s.client()
	if status := elsewhere.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("second sign-in = %d", status)
	}
	third := s.client()
	if status := third.login(superEmail, superPassword); status != http.StatusOK {
		t.Fatalf("third sign-in = %d", status)
	}

	var list ownSessions
	here.must(http.StatusOK, http.MethodGet, "/sessions", nil, &list)

	current, others := "", []string{}
	for _, session := range list.Sessions {
		if session.Current {
			current = session.ID
		} else if session.Active {
			others = append(others, session.ID)
		}
	}
	if current == "" || len(others) != 2 {
		t.Fatalf("sessions = %+v, want one current and two others", list.Sessions)
	}

	if status := here.do(http.MethodDelete, "/sessions/"+current, nil, nil); status != http.StatusConflict {
		t.Errorf("ending the current session = %d, want 409", status)
	}
	if status := here.do(http.MethodDelete, "/sessions/not-an-id", nil, nil); status != http.StatusNotFound {
		t.Errorf("ending a session that is not one = %d, want 404", status)
	}

	// One of the others, by id: it is signed out, and the rest are not.
	here.must(http.StatusNoContent, http.MethodDelete, "/sessions/"+others[0], nil, nil)
	signedOut := 0
	for _, c := range []*client{elsewhere, third} {
		if c.do(http.MethodGet, "/me", nil, nil) == http.StatusUnauthorized {
			signedOut++
		}
	}
	if signedOut != 1 {
		t.Errorf("after ending one session, %d of the other two were signed out, want 1", signedOut)
	}

	// Ending it again finds nothing to end.
	if status := here.do(http.MethodDelete, "/sessions/"+others[0], nil, nil); status != http.StatusNotFound {
		t.Errorf("ending an ended session = %d, want 404", status)
	}

	// Everywhere else: the last other browser goes, this one stays.
	var ended struct {
		Ended int `json:"ended"`
	}
	here.must(http.StatusOK, http.MethodDelete, "/sessions", nil, &ended)
	if ended.Ended != 1 {
		t.Errorf("ended = %d, want the one other session left", ended.Ended)
	}
	if status := here.do(http.MethodGet, "/me", nil, nil); status != http.StatusOK {
		t.Errorf("the session that signed out everywhere else = %d, want still signed in", status)
	}
	for _, c := range []*client{elsewhere, third} {
		if status := c.do(http.MethodGet, "/me", nil, nil); status != http.StatusUnauthorized {
			t.Errorf("another session after signing out everywhere else = %d, want 401", status)
		}
	}
}

// An administrator's picture is saved with their profile, as a full address
// or nothing, and comes back with who they are.
func TestLiveAdminAvatar(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	profile := func(avatar string) map[string]string {
		return map[string]string{"first_name": "Root", "email": superEmail, "avatar_url": avatar}
	}

	if status := super.do(http.MethodPatch, "/me", profile("javascript:alert(1)"), nil); status != http.StatusBadRequest {
		t.Errorf("an avatar that could run something = %d, want 400", status)
	}

	super.must(http.StatusOK, http.MethodPatch, "/me", profile("https://example.com/me.png"), nil)

	var me struct {
		Admin struct {
			AvatarURL string `json:"avatar_url"`
		} `json:"admin"`
	}
	super.must(http.StatusOK, http.MethodGet, "/me", nil, &me)
	if me.Admin.AvatarURL != "https://example.com/me.png" {
		t.Errorf("avatar_url = %q, want the saved address", me.Admin.AvatarURL)
	}

	super.must(http.StatusOK, http.MethodPatch, "/me", profile(""), nil)
	super.must(http.StatusOK, http.MethodGet, "/me", nil, &me)
	if me.Admin.AvatarURL != "" {
		t.Errorf("avatar_url = %q, want it cleared", me.Admin.AvatarURL)
	}
}

type logPage struct {
	Logs []struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	} `json:"logs"`
	Next string `json:"next"`
}

// The log is read a page at a time, newest first, and narrowed on the
// server: by kind of entry, by text, by day. Paging never repeats or skips
// an entry, and the export writes what the filters match.
func TestLiveLogFilters(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	// A few sign-ins that failed and one profile change, beside the sign-in
	// the setup made.
	for range 3 {
		s.client().login(superEmail, "not-the-password")
	}
	super.must(http.StatusOK, http.MethodPatch, "/me", map[string]string{
		"first_name": "Rooted", "email": superEmail,
	}, nil)

	var all logPage
	super.must(http.StatusOK, http.MethodGet, "/logs?limit=200", nil, &all)
	if len(all.Logs) < 5 {
		t.Fatalf("logs = %d entries, want at least five", len(all.Logs))
	}

	// Two to a page, followed to the end, is the whole log in order.
	seen := []string{}
	for path := "/logs?limit=2"; path != ""; {
		var page logPage
		super.must(http.StatusOK, http.MethodGet, path, nil, &page)
		for _, entry := range page.Logs {
			seen = append(seen, entry.ID)
		}

		path = ""
		if page.Next != "" {
			path = "/logs?limit=2&before=" + page.Next
		}
	}
	if len(seen) != len(all.Logs) {
		t.Fatalf("paged through %d entries, want %d", len(seen), len(all.Logs))
	}
	for i := range seen {
		if seen[i] != all.Logs[i].ID {
			t.Fatalf("page order differs at %d", i)
		}
	}

	var failed logPage
	super.must(http.StatusOK, http.MethodGet, "/logs?action=admin.login_failed", nil, &failed)
	if len(failed.Logs) != 3 {
		t.Errorf("failed sign-ins = %d, want 3", len(failed.Logs))
	}
	for _, entry := range failed.Logs {
		if entry.Action != "admin.login_failed" {
			t.Errorf("an action filter let through %q", entry.Action)
		}
	}

	var searched logPage
	super.must(http.StatusOK, http.MethodGet, "/logs?q=PROFILE_upd", nil, &searched)
	if len(searched.Logs) != 1 || searched.Logs[0].Action != "admin.profile_updated" {
		t.Errorf("search = %+v, want the one profile change", searched.Logs)
	}

	// A day long ago has nothing in it.
	var old logPage
	super.must(http.StatusOK, http.MethodGet, "/logs?from=2000-01-01&to=2000-01-02", nil, &old)
	if len(old.Logs) != 0 {
		t.Errorf("entries on a day in 2000 = %d, want none", len(old.Logs))
	}

	for _, bad := range []string{"from=yesterday", "before=nonsense", "action=DROP TABLE"} {
		if status := super.do(http.MethodGet, "/logs?"+bad, nil, nil); status != http.StatusBadRequest {
			t.Errorf("filter %q = %d, want 400", bad, status)
		}
	}

	csv, status := super.raw(http.MethodGet, "/logs/export?action=admin.login_failed")
	if status != http.StatusOK {
		t.Fatalf("export = %d, want 200", status)
	}
	lines := strings.Split(strings.TrimSpace(csv), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[0], "time,actor,action") {
		t.Errorf("export = %d lines starting %q, want a header and three rows", len(lines), lines[0])
	}
}

// An export whose first read fails answers with an error, not with an empty
// file that would look like there was nothing to export.
func TestLiveAnExportThatCannotReadAnswersAnError(t *testing.T) {
	s := newLiveServer(t)
	super := s.superAdmin()

	// Signing in wrote to the log; only now is it taken away.
	if err := s.db.Exec("ALTER TABLE audit_logs RENAME TO audit_logs_gone").Error; err != nil {
		t.Fatal(err)
	}

	body, status := super.raw(http.MethodGet, "/logs/export")
	if status != http.StatusInternalServerError {
		t.Errorf("export = %d, want 500", status)
	}
	if strings.HasPrefix(body, "time,actor") {
		t.Error("the failed export still began a CSV file")
	}
}
