package oidc

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// suffixOf is what the range API lists for a password: its SHA-1 after the
// five characters the request names.
func suffixOf(password string) string {
	sum := sha1.Sum([]byte(password))
	return strings.ToUpper(hex.EncodeToString(sum[:]))[5:]
}

// The check lets a password through whenever it cannot say, and says why in
// the log, so a failing API never stops anybody choosing a password.
func TestThePasswordBreachCheck(t *testing.T) {
	const password = "correct horse battery staple"
	listed := suffixOf(password) + ":42\r\n"

	tests := []struct {
		name    string
		answer  func(w http.ResponseWriter)
		want    bool
		wantLog string
	}{
		{
			name:   "a password in the range",
			answer: func(w http.ResponseWriter) { fmt.Fprint(w, "0000000000000000000000000000000000A:3\r\n"+listed) },
			want:   true,
		},
		{
			name:   "a password only as padding, counted 0",
			answer: func(w http.ResponseWriter) { fmt.Fprint(w, suffixOf(password)+":0\r\n") },
		},
		{
			name:   "a password not in the range",
			answer: func(w http.ResponseWriter) { fmt.Fprint(w, "0000000000000000000000000000000000A:3\r\n") },
		},
		{
			name:   "the API failing",
			answer: func(w http.ResponseWriter) { w.WriteHeader(http.StatusServiceUnavailable) },
		},
		{
			name: "an answer cut off before the password",
			answer: func(w http.ResponseWriter) {
				// Promised longer than it is, then the connection drops.
				w.Header().Set("Content-Length", "4096")
				fmt.Fprint(w, "0000000000000000000000000000000000A:3\r\n")
				w.(http.Flusher).Flush()
				conn, _, err := w.(http.Hijacker).Hijack()
				if err == nil {
					_ = conn.Close()
				}
			},
			wantLog: "cut off",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { tt.answer(w) }))
			defer api.Close()

			var logged bytes.Buffer
			s := &Service{
				pwnedURL:     api.URL,
				breachClient: api.Client(),
				log:          slog.New(slog.NewTextHandler(&logged, nil)),
			}

			if got := s.PasswordBreached(t.Context(), password); got != tt.want {
				t.Errorf("PasswordBreached = %v, want %v", got, tt.want)
			}
			if tt.wantLog != "" && !strings.Contains(logged.String(), tt.wantLog) {
				t.Errorf("logged %q, want a warning that the answer was %s", logged.String(), tt.wantLog)
			}
		})
	}
}
