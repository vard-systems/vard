package clientcert

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"vard/internal/brand"
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

// The header is only as good as the proxy in front: on every site but the
// mutual TLS one, nginx must drop a certificate header a client wrote
// itself — or anyone could claim any certificate, and certificates are
// public. On the mutual TLS site it must be the connection's own. nginx
// passes a client's headers on by default, so every location that proxies
// says which.
func TestNginxPassesOnlyTheConnectionsCertificate(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "deploy", "nginx", brand.Slug+".conf.template"))
	if err != nil {
		t.Fatal(err)
	}

	servers := strings.Split(string(template), "\nserver {")[1:]
	mutual := 0
	for _, server := range servers {
		name := regexp.MustCompile(`server_name ([^;]+);`).FindStringSubmatch(server)[1]
		asksForCertificates := strings.Contains(server, "ssl_verify_client")
		if asksForCertificates {
			mutual++
		}

		for _, location := range strings.Split(server, "location ")[1:] {
			if !strings.Contains(location, "proxy_pass") {
				continue
			}

			header := regexp.MustCompile(`proxy_set_header `+Header+` ([^;]+);`).FindAllStringSubmatch(location, -1)
			switch {
			case len(header) != 1:
				t.Errorf("%s: a location passes on to an app without setting %s once", name, Header)
			case asksForCertificates && header[0][1] != "$ssl_client_escaped_cert":
				t.Errorf("%s: the mutual TLS site sets %s to %s, want the connection's certificate", name, Header, header[0][1])
			case !asksForCertificates && header[0][1] != `""`:
				t.Errorf("%s: sets %s to %s, want it cleared", name, Header, header[0][1])
			}
		}
	}

	if mutual != 1 {
		t.Errorf("%d sites ask for client certificates, want exactly the mutual TLS one", mutual)
	}
}
