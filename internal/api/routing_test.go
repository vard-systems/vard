package api

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// A browser never reaches the API directly: each app's server answers its
// origin and passes on only the paths in its API_PATHS
// (web/<app>/src/lib/server/api.ts). A route outside them would reach the
// API's tests and nobody else, so the list is held to the routes here.
func TestEveryRouteIsReachableThroughItsApp(t *testing.T) {
	tests := []struct {
		name   string
		engine *gin.Engine
		source string
	}{
		{"the public API, through the id app", testEngine(), "../../web/id/src/lib/server/api.ts"},
		{"the admin API, through the console", testAdminEngine(), "../../web/console/src/lib/server/api.ts"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths := apiPaths(t, read(t, test.source))

			for _, route := range test.engine.Routes() {
				// The orchestrator's, asked on the API's own port.
				if route.Path == "/healthz" {
					continue
				}

				if !anyPrefix(paths, concrete(route.Path)) {
					t.Errorf("%s %s is not passed on to the API by %s", route.Method, route.Path, test.source)
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

// apiPaths reads the `const API_PATHS = [...]` list of an app's api.ts.
func apiPaths(t *testing.T, source string) []string {
	t.Helper()

	list := regexp.MustCompile(`(?s)const API_PATHS = \[(.*?)\]`).FindStringSubmatch(source)
	if list == nil {
		t.Fatal("the file has no `const API_PATHS = [...]`")
	}

	var paths []string
	for _, quoted := range regexp.MustCompile(`'([^']+)'`).FindAllStringSubmatch(list[1], -1) {
		paths = append(paths, quoted[1])
	}

	return paths
}

// anyPrefix matches as isApiPath does: the request path is one, or starts
// with one followed by a slash.
func anyPrefix(prefixes []string, path string) bool {
	for _, prefix := range prefixes {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}

	return false
}
