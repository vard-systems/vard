package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// The limit is what keeps a caller from having a body read into memory until
// there is none left, which takes no account and no session to try. A body
// that says its length is refused before any of it is read; one that does not
// is cut short as the handler reads it.
func TestBodyLimit(t *testing.T) {
	const max = 64

	tests := []struct {
		name     string
		body     string
		unstated bool
		want     int
		code     string
		read     bool
	}{
		{name: "a body that fits", body: strings.Repeat("a", max), want: http.StatusNoContent, read: true},
		{name: "no body at all", want: http.StatusNoContent, read: true},
		{
			name: "a body that says it is too long",
			body: strings.Repeat("a", max+1),
			want: http.StatusRequestEntityTooLarge,
			code: "body_too_large",
		},
		{
			name:     "a body that does not say how long it is",
			body:     strings.Repeat("a", max+1),
			unstated: true,
			want:     http.StatusBadRequest,
			read:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reached := false

			r := gin.New()
			r.Use(BodyLimit(max))
			r.POST("/thing", func(c *gin.Context) {
				reached = true
				if _, err := io.ReadAll(c.Request.Body); err != nil {
					// What a handler binding a body reports: it could not be
					// read.
					c.Status(http.StatusBadRequest)
					return
				}
				c.Status(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodPost, "/thing", strings.NewReader(tt.body))
			if tt.unstated {
				req.ContentLength = -1
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tt.want, w.Body)
			}
			if reached != tt.read {
				t.Errorf("the handler ran = %v, want %v", reached, tt.read)
			}
			if tt.code != "" {
				var answer struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &answer); err != nil {
					t.Fatalf("the answer is not JSON: %s", w.Body)
				}
				if answer.Code != tt.code {
					t.Errorf("code = %q, want %q", answer.Code, tt.code)
				}
			}
		})
	}
}
