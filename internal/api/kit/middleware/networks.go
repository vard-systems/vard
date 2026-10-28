package middleware

import (
	"fmt"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"

	"vard/internal/api/kit/respond"
)

var networkRefused = respond.Define(http.StatusForbidden, "admin_network_refused", respond.Admin)

// Networks answers only callers inside one of the networks, by Gin's
// ClientIP, which believes X-Forwarded-For only from the trusted proxies.
// None lets everyone through.
func Networks(cidrs []string) (gin.HandlerFunc, error) {
	allowed := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("admin networks: %q: %w", cidr, err)
		}
		allowed = append(allowed, network)
	}

	return func(c *gin.Context) {
		if len(allowed) == 0 {
			c.Next()
			return
		}

		caller := net.ParseIP(c.ClientIP())
		for _, network := range allowed {
			if caller != nil && network.Contains(caller) {
				c.Next()
				return
			}
		}

		respond.Abort(c, networkRefused)
	}, nil
}
