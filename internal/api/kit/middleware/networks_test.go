package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNetworksAnswerOnlyTheirCallers(t *testing.T) {
	cases := []struct {
		name     string
		networks []string
		caller   string
		want     int
	}{
		{"no networks answer everyone", nil, "203.0.113.9", http.StatusOK},
		{"a caller inside a network", []string{"10.0.0.0/8", "203.0.113.0/24"}, "203.0.113.9", http.StatusOK},
		{"a caller outside every network", []string{"10.0.0.0/8"}, "203.0.113.9", http.StatusForbidden},
		{"an IPv6 caller inside", []string{"2001:db8::/32"}, "2001:db8::1", http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			guard, err := Networks(c.networks)
			if err != nil {
				t.Fatal(err)
			}

			r := gin.New()
			r.GET("/", guard, func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = c.caller + ":1234"
			if strings.Contains(c.caller, ":") {
				req.RemoteAddr = "[" + c.caller + "]:1234"
			}
			res := httptest.NewRecorder()
			r.ServeHTTP(res, req)

			if res.Code != c.want {
				t.Errorf("a call from %s = %d, want %d", c.caller, res.Code, c.want)
			}
		})
	}

	if _, err := Networks([]string{"not a network"}); err == nil {
		t.Error("a list holding something other than a CIDR was taken")
	}
}
