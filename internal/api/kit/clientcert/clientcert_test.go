package clientcert

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"vard/internal/model"
	"vard/internal/model/certtest"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// The certificate header is the apps' word that the caller holds the
// certificate's key. Believed from anyone else, or with mutual TLS off, it
// would let a caller claim any certificate it has a copy of — and
// certificates are public.
func TestTheCertificateIsBelievedOnlyFromTheApps(t *testing.T) {
	cert := certtest.SelfSigned(t, "shop")

	tests := []struct {
		name    string
		enabled bool
		from    string
		header  string
		want    string
	}{
		{"from the id app, with mutual TLS on", true, "172.30.0.2:41000", Header + ":" + certtest.Header(cert), model.CertificateThumbprint(cert)},
		{"from loopback under make dev", true, "127.0.0.1:41000", Header + ":" + certtest.Header(cert), model.CertificateThumbprint(cert)},
		{"from a caller reaching the API itself", true, "203.0.113.7:41000", Header + ":" + certtest.Header(cert), ""},
		{"from another container on the network", true, "172.30.0.200:41000", Header + ":" + certtest.Header(cert), ""},
		{"with mutual TLS off", false, "172.30.0.2:41000", Header + ":" + certtest.Header(cert), ""},
		{"that is not a certificate", true, "172.30.0.2:41000", Header + ":" + "not%20a%20certificate", ""},
		{"none at all", true, "172.30.0.2:41000", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware, err := New(tt.enabled, []string{"172.30.0.2/32", "172.30.0.3", "127.0.0.1/32", "::1/128"})
			if err != nil {
				t.Fatal(err)
			}

			var got string
			r := gin.New()
			r.GET("/oauth2/userinfo", middleware, func(c *gin.Context) { got = Thumbprint(c) })

			req := httptest.NewRequest(http.MethodGet, "/oauth2/userinfo", nil)
			req.RemoteAddr = tt.from
			if tt.header != "" {
				name, value, _ := strings.Cut(tt.header, ":")
				req.Header.Set(name, value)
			}
			r.ServeHTTP(httptest.NewRecorder(), req)

			if got != tt.want {
				t.Errorf("thumbprint = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestATrustedProxyThatIsNoAddressIsRefused(t *testing.T) {
	if _, err := New(true, []string{"not-an-address"}); err == nil {
		t.Error("New() = nil, want an error: a mistyped proxy must not quietly trust nobody, or everybody")
	}
}
