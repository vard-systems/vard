package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vard/internal/brand"
)

// set runs one writer and returns the cookies it set, by name.
func set(write func(c *gin.Context)) map[string]*http.Cookie {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	write(c)

	out := map[string]*http.Cookie{}
	for _, cookie := range (&http.Response{Header: w.Header()}).Cookies() {
		out[cookie.Name] = cookie
	}
	return out
}

// Over https every session and sign-in cookie is a __Host- one: Secure, the
// whole host's path, no Domain, so a sibling subdomain can neither plant nor
// overwrite it; the copy under the old name is cleared as it is set. Over
// plain http, where a browser would refuse the prefix, the names are as they
// were.
func TestSessionCookiesAreHostOnlyOverHTTPS(t *testing.T) {
	writers := []struct {
		name  string
		base  string
		path  string
		write func(c *gin.Context, secure bool)
	}{
		{"the administrator's session", Cookie, "/", func(c *gin.Context, secure bool) { Set(c, "token", secure) }},
		{"the user's session", UserCookie, "/", func(c *gin.Context, secure bool) { SetUser(c, "token", time.Now().Add(time.Hour), true, secure) }},
		{"the sign-in state", SignInStateCookie, signInStatePath, func(c *gin.Context, secure bool) { SetSignInState(c, "state", secure) }},
		{"the second factor", SecondFactorCookie, secondFactorPath, func(c *gin.Context, secure bool) { SetSecondFactor(c, "handle", time.Now().Add(time.Minute), secure) }},
		{"the known device", KnownDeviceCookie, knownDevicePath, func(c *gin.Context, secure bool) { SetKnownDevice(c, "device", secure) }},
	}

	for _, w := range writers {
		t.Run(w.name+" over https", func(t *testing.T) {
			cookies := set(func(c *gin.Context) { w.write(c, true) })

			prefixed := cookies[brand.HostPrefix+w.base]
			if prefixed == nil {
				t.Fatalf("no %s%s set; got %v", brand.HostPrefix, w.base, cookies)
			}
			if !prefixed.Secure || prefixed.Path != "/" || prefixed.Domain != "" || !prefixed.HttpOnly {
				t.Errorf("%s = Secure %v, Path %q, Domain %q, HttpOnly %v; want Secure, /, none, HttpOnly",
					prefixed.Name, prefixed.Secure, prefixed.Path, prefixed.Domain, prefixed.HttpOnly)
			}
			if old := cookies[w.base]; old == nil || old.MaxAge >= 0 {
				t.Errorf("the old %s was not cleared: %v", w.base, old)
			}
		})

		t.Run(w.name+" over http", func(t *testing.T) {
			cookies := set(func(c *gin.Context) { w.write(c, false) })

			if cookies[brand.HostPrefix+w.base] != nil {
				t.Error("a __Host- cookie over plain http, which a browser refuses")
			}
			if plain := cookies[w.base]; plain == nil || plain.Path != w.path {
				t.Errorf("%s = %v, want it set at %s", w.base, plain, w.path)
			}
		})
	}
}

// A session is read from its prefixed cookie, and for one release from the
// old name when there is none; two of either count as none.
func TestSessionCookiesAreReadPrefixedFirst(t *testing.T) {
	cases := []struct {
		name    string
		cookies []string
		want    string
	}{
		{"the prefixed one", []string{brand.HostPrefix + UserCookie + "=new"}, "new"},
		{"the old name, before the prefix", []string{UserCookie + "=old"}, "old"},
		{"the prefixed one over the old", []string{UserCookie + "=old", brand.HostPrefix + UserCookie + "=new"}, "new"},
		{"two of the old, one planted", []string{UserCookie + "=old", UserCookie + "=planted"}, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Cookie", strings.Join(tt.cookies, "; "))

			if got := UserToken(c); got != tt.want {
				t.Errorf("UserToken = %q, want %q", got, tt.want)
			}
		})
	}
}
