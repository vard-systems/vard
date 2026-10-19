package csrf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

const app = "https://id.mywebsite.com"

func TestCSRF(t *testing.T) {
	r := gin.New()
	r.Use(New([]string{app}))
	r.Any("/change", func(c *gin.Context) { c.Status(http.StatusNoContent) })

	cases := []struct {
		name        string
		method      string
		body        string
		contentType string
		headers     map[string]string
		want        int
	}{
		{name: "the app itself", method: http.MethodPost, body: `{}`, contentType: "application/json", headers: map[string]string{"Origin": app}, want: http.StatusNoContent},
		{name: "json with charset", method: http.MethodPatch, body: `{}`, contentType: "application/json; charset=utf-8", headers: map[string]string{"Origin": app}, want: http.StatusNoContent},
		{name: "no body, same origin", method: http.MethodPost, headers: map[string]string{"Origin": app, "Sec-Fetch-Site": "same-origin"}, want: http.StatusNoContent},
		{name: "a sibling subdomain", method: http.MethodPost, body: `{}`, contentType: "application/json", headers: map[string]string{"Origin": "https://blog.mywebsite.com"}, want: http.StatusForbidden},
		{name: "a form posting text/plain", method: http.MethodPost, body: `{"email":"x"}`, contentType: "text/plain", headers: map[string]string{"Origin": app}, want: http.StatusUnsupportedMediaType},
		{name: "a form with no content type", method: http.MethodPost, body: `a=b`, headers: map[string]string{"Origin": "https://evil.example"}, want: http.StatusUnsupportedMediaType},
		{name: "cross-site without Origin", method: http.MethodDelete, headers: map[string]string{"Sec-Fetch-Site": "same-site"}, want: http.StatusForbidden},
		{name: "not a browser", method: http.MethodPost, body: `{}`, contentType: "application/json", want: http.StatusNoContent},
		{name: "a bearer token from an allowed page on another origin", method: http.MethodPatch, body: `{}`, contentType: "application/json", headers: map[string]string{"Origin": "https://shop.example", "Authorization": "Bearer abc"}, want: http.StatusNoContent},
		{name: "a bearer token still sends JSON", method: http.MethodPost, body: `a=b`, contentType: "text/plain", headers: map[string]string{"Authorization": "Bearer abc"}, want: http.StatusUnsupportedMediaType},
		{name: "an empty bearer is no token", method: http.MethodPost, body: `{}`, contentType: "application/json", headers: map[string]string{"Origin": "https://evil.example", "Authorization": "Bearer "}, want: http.StatusForbidden},
		{name: "reads are not checked", method: http.MethodGet, headers: map[string]string{"Origin": "https://evil.example"}, want: http.StatusNoContent},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/change", strings.NewReader(tc.body))
			if tc.body == "" {
				req = httptest.NewRequest(tc.method, "/change", nil)
			}
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tc.want {
				t.Errorf("status = %d, want %d (%s)", w.Code, tc.want, w.Body)
			}
		})
	}
}
