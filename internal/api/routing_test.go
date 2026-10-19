package api

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Each app calls its API on its own origin. The binary serves both on the
// API's listener, so there every route is reachable; under `make dev` the
// Vite dev server answers that origin and passes on only the paths in its
// config's apiPaths. A route outside them would reach the API's tests and
// nobody in development, so the list is held to the routes here.
func TestEveryRouteIsReachableThroughItsApp(t *testing.T) {
	tests := []struct {
		name   string
		engine *gin.Engine
		config string
	}{
		{"the public API, through the id app", testEngine(), "../../web/id/vite.config.ts"},
		{"the admin API, through the console", testAdminEngine(), "../../web/console/vite.config.ts"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths := viteAPIPaths(t, read(t, test.config))

			for _, route := range test.engine.Routes() {
				// The orchestrator's, asked on the API's own port.
				if route.Path == "/healthz" {
					continue
				}

				if !anyPrefix(paths, concrete(route.Path)) {
					t.Errorf("%s %s is not passed on to the API by %s", route.Method, route.Path, test.config)
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

// viteAPIPaths reads the `const apiPaths = [...]` list of a vite.config.ts.
func viteAPIPaths(t *testing.T, config string) []string {
	t.Helper()

	list := regexp.MustCompile(`(?s)const apiPaths = \[(.*?)\]`).FindStringSubmatch(config)
	if list == nil {
		t.Fatal("the config has no `const apiPaths = [...]`")
	}

	var paths []string
	for _, quoted := range regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(list[1], -1) {
		paths = append(paths, quoted[1])
	}

	return paths
}

// anyPrefix matches as Vite's proxy does for a key that is not a regular
// expression: the request path starts with it.
func anyPrefix(prefixes []string, path string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
