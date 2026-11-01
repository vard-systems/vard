package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"vard/core/brand"
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
// overwrite it. Over plain http, where a browser would refuse the prefix,
// each keeps its bare name.
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
			if bare := cookies[w.base]; bare != nil {
				t.Errorf("a bare %s was set beside the prefixed one", w.base)
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

// A session is read under the one name its listener writes it with: the
// __Host- one over https, the bare one over http. Under the other name it is
// none, and so are two of it.
func TestSessionCookiesAreReadUnderTheirListenersName(t *testing.T) {
	cases := []struct {
		name    string
		secure  bool
		cookies []string
		want    string
	}{
		{"over https, the prefixed one", true, []string{brand.HostPrefix + UserCookie + "=new"}, "new"},
		{"over https, a bare one planted by a sibling", true, []string{UserCookie + "=planted"}, ""},
		{"over http, the bare one", false, []string{UserCookie + "=plain"}, "plain"},
		{"over http, two of it", false, []string{UserCookie + "=one", UserCookie + "=two"}, ""},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Cookie", strings.Join(tt.cookies, "; "))
			CookieMode(tt.secure)(c)

			if got := UserToken(c); got != tt.want {
				t.Errorf("UserToken = %q, want %q", got, tt.want)
			}
		})
	}
}

// The sign-in pages write the reader's language under the __Host- name over
// https; the API reads it under the same one, so emails come in that language.
func TestTheLanguageIsReadUnderTheNameThePagesWrite(t *testing.T) {
	cases := []struct {
		name   string
		secure bool
		cookie string
		want   string
	}{
		{"over https, the prefixed one", true, brand.HostPrefix + brand.LanguageCookie + "=fr", "fr"},
		{"over https, a bare one planted by a sibling", true, brand.LanguageCookie + "=xx", ""},
		{"over http, the bare one", false, brand.LanguageCookie + "=fr", "fr"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Cookie", tt.cookie)
			CookieMode(tt.secure)(c)

			if got := Language(c); got != tt.want {
				t.Errorf("Language = %q, want %q", got, tt.want)
			}
		})
	}
}
