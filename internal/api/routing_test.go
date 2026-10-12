package api

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// A browser never reaches the API directly: the id and console apps' origins
// are in front of it, and a path reaches the API only where something on that
// origin routes it there — Vite's proxy in development, Caddy in a
// deployment. A route outside those prefixes answers the API's tests and
// nobody else, so every route has to be routed by both.
func TestEveryRouteIsReachableThroughItsApp(t *testing.T) {
	caddy := read(t, "../../deploy/Caddyfile.local")

	tests := []struct {
		name   string
		engine *gin.Engine
		vite   string
		site   string
	}{
		{"the public API, through the id app", testEngine(), "../../web/id/vite.config.ts", "id.localhost"},
		{"the admin API, through the console", testAdminEngine(), "../../web/console/vite.config.ts", "admin-id.localhost"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vite := viteProxyPrefixes(t, read(t, test.vite))
			edge := caddyAPIPaths(t, caddy, test.site)

			for _, route := range test.engine.Routes() {
				// The orchestrator's, asked on the API's own port.
				if route.Path == "/healthz" {
					continue
				}

				path := concrete(route.Path)
				if !anyPrefix(vite, path) {
					t.Errorf("%s %s is not in the proxy of %s", route.Method, route.Path, test.vite)
				}
				if !anyCaddyPath(edge, path) {
					t.Errorf("%s %s is not routed to the API by the %s site of Caddyfile.local", route.Method, route.Path, test.site)
				}
			}
		})
	}
}

func read(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(data)
}

// concrete fills a route's parameters in, giving a path a request could have.
func concrete(route string) string {
	return regexp.MustCompile(`[:*][a-z_]+`).ReplaceAllString(route, "x")
}

// viteProxyPrefixes reads the keys of `const proxy = { … }`, which Vite
// matches as prefixes of the path.
func viteProxyPrefixes(t *testing.T, config string) []string {
	t.Helper()

	block := regexp.MustCompile(`(?s)const proxy = \{(.*?)\};`).FindStringSubmatch(config)
	if block == nil {
		t.Fatal("no `const proxy = { … };` in the Vite config")
	}

	var prefixes []string
	for _, key := range regexp.MustCompile(`'(/[^']*)'\s*:`).FindAllStringSubmatch(block[1], -1) {
		prefixes = append(prefixes, key[1])
	}
	if len(prefixes) == 0 {
		t.Fatal("the Vite proxy routes nothing")
	}

	return prefixes
}

// caddyAPIPaths reads the `@api path …` matcher of one site block.
func caddyAPIPaths(t *testing.T, caddyfile, site string) []string {
	t.Helper()

	block := regexp.MustCompile(`(?ms)^` + regexp.QuoteMeta(site) + ` \{\n(.*?)^\}`).FindStringSubmatch(caddyfile)
	if block == nil {
		t.Fatalf("no %s site in the Caddyfile", site)
	}

	matcher := regexp.MustCompile(`(?m)^\s*@api path (.+)$`).FindStringSubmatch(block[1])
	if matcher == nil {
		t.Fatalf("the %s site has no `@api path` matcher", site)
	}

	return strings.Fields(matcher[1])
}

func anyPrefix(prefixes []string, path string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

// anyCaddyPath matches as Caddy's path matcher does for these patterns: exact,
// or a prefix where the pattern ends in *.
func anyCaddyPath(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if prefix, wildcard := strings.CutSuffix(pattern, "*"); wildcard && strings.HasPrefix(path, prefix) || pattern == path {
			return true
		}
	}

	return false
}
