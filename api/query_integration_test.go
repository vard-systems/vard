package api

import (
	"net/http"
	"net/url"
	"testing"
)

// The queries rewritten to be served by an index still answer what they
// answered before — user search as one expression over the whole record, and
// an API's activity read through its two indexes.

func TestLiveUserSearch(t *testing.T) {
	f := newOAuthFixture(t)

	for _, body := range []map[string]any{
		{"email": "grace@example.com", "first_name": "Grace", "last_name": "Hopper"},
		{"email": "percent@example.com", "first_name": "Per", "last_name": "Cent"},
	} {
		body["password"], body["confirm_password"] = "a-password-123", "a-password-123"
		f.super.must(http.StatusCreated, http.MethodPost, "/users", body, nil)
	}

	tests := []struct {
		name, search string
		want         []string
	}{
		{"part of an address", "grace@", []string{"grace@example.com"}},
		{"a last name, in any case", "HOPPER", []string{"grace@example.com"}},
		{"a name typed in full, across two fields", "grace hopper", []string{"grace@example.com"}},
		{"a wildcard is only itself", "%", nil},
		{"an underscore is only itself", "_", nil},
		{"the fixture's user still", "lovelace", []string{adaEmail}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var page struct {
				Users []struct {
					Email string `json:"email"`
				} `json:"users"`
				Total int64 `json:"total"`
			}
			f.super.must(http.StatusOK, http.MethodGet, "/users?search="+url.QueryEscape(tt.search), nil, &page)

			got := []string{}
			for _, user := range page.Users {
				got = append(got, user.Email)
			}
			if len(got) != len(tt.want) || int(page.Total) != len(tt.want) {
				t.Fatalf("search %q = %v (total %d), want %v", tt.search, got, page.Total, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("search %q = %v, want %v", tt.search, got, tt.want)
				}
			}
		})
	}
}

func TestLiveAPIActivity(t *testing.T) {
	f := newOAuthFixture(t)

	// The fixture made the API and authorized the shop for it: one entry
	// targets the API, the other names it in its metadata.
	var logs struct {
		Logs []struct {
			Action string `json:"action"`
		} `json:"logs"`
	}
	f.super.must(http.StatusOK, http.MethodGet, "/apis/"+f.apiID+"/logs", nil, &logs)

	seen := map[string]bool{}
	for _, entry := range logs.Logs {
		seen[entry.Action] = true
	}
	for _, action := range []string{"api.created", "application.api_authorized"} {
		if !seen[action] {
			t.Errorf("the API's activity has no %s: %+v", action, logs.Logs)
		}
	}
}
