// Package clientcert reads the TLS client certificate a request came with.
// TLS ends at the proxy in front of the apps, which asks for a certificate
// on the mutual TLS host and passes it on, URL-encoded PEM, in Header; the id
// app passes it to the API. It is believed only when mutual TLS is on and
// only from the trusted proxies, as X-Forwarded-For is: from anyone else it
// would let a caller claim a certificate whose key it does not hold.
package clientcert

import (
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"vard/internal/model"
)

const (
	Header = "X-Client-Certificate"

	contextKey = "client_certificate"
	// maxHeader bounds what is parsed: a client certificate is a few KB.
	maxHeader = 16 << 10
)

// New reads the certificate when enabled, from requests whose connection
// comes from one of trusted (addresses or CIDR ranges).
func New(enabled bool, trusted []string) (gin.HandlerFunc, error) {
	ranges := make([]*net.IPNet, 0, len(trusted))
	for _, entry := range trusted {
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil && ip.To4() != nil {
				entry += "/32"
			} else {
				entry += "/128"
			}
		}

		_, network, err := net.ParseCIDR(entry)
		if err != nil {
			return nil, fmt.Errorf("trusted proxy %q: %w", entry, err)
		}
		ranges = append(ranges, network)
	}

	return func(c *gin.Context) {
		if enabled && fromTrusted(c, ranges) {
			if cert := parse(c.GetHeader(Header)); cert != nil {
				c.Set(contextKey, cert)
			}
		}
		c.Next()
	}, nil
}

// From is the request's client certificate, nil when it came with none.
func From(c *gin.Context) *x509.Certificate {
	value, _ := c.Get(contextKey)
	cert, _ := value.(*x509.Certificate)
	return cert
}

// Thumbprint is the x5t#S256 of the request's certificate, "" when none.
func Thumbprint(c *gin.Context) string {
	if cert := From(c); cert != nil {
		return model.CertificateThumbprint(cert)
	}
	return ""
}

func fromTrusted(c *gin.Context, ranges []*net.IPNet) bool {
	ip := net.ParseIP(c.RemoteIP())
	if ip == nil {
		return false
	}

	for _, network := range ranges {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// parse reads nginx's $ssl_client_escaped_cert. PathUnescape, not
// QueryUnescape: a + in the base64 is a +, not a space.
func parse(header string) *x509.Certificate {
	if header == "" || len(header) > maxHeader {
		return nil
	}

	text, err := url.PathUnescape(header)
	if err != nil {
		return nil
	}

	cert, err := model.ParseCertificate(text)
	if err != nil {
		return nil
	}
	return cert
}
