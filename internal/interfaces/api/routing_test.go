package api

import (
	"io/fs"
	"os"
	"path/filepath"
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
		{"the public API, through the id app", testEngine(), "../../../web/id/src/lib/server/api.ts"},
		{"the admin API, through the console", testAdminEngine(), "../../../web/console/src/lib/server/api.ts"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			paths := apiPaths(t, read(t, test.source))

			for _, route := range test.engine.Routes() {
				// The orchestrator's, asked on the API's own port.
				if route.Path == "/healthz" || route.Path == "/readyz" {
					continue
				}

				if !anyPrefix(paths, concrete(route.Path)) {
					t.Errorf("%s %s is not passed on to the API by %s", route.Method, route.Path, test.source)
				}
			}
		})
	}
}

// The provider takes form posts from other origins and from none: a backend
// redeeming a code, an SPA at the token endpoint, a SAML or Apple answer. A
// built SvelteKit app refuses those before its hooks run, so the id app turns
// that check off — which is safe only while the app has no form action or
// endpoint of its own for the check to protect.
func TestTheIDAppLetsTheProviderTakeFormPosts(t *testing.T) {
	config := read(t, "../../../web/id/vite.config.ts")
	if !regexp.MustCompile(`csrf:\s*\{\s*trustedOrigins:\s*\['\*'\]\s*\}`).MatchString(config) {
		t.Error("web/id/vite.config.ts does not set csrf: { trustedOrigins: ['*'] }; the token endpoint answers 403 through the built app")
	}

	err := filepath.WalkDir("../../../web/id/src/routes", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		switch {
		case strings.HasPrefix(name, "+server."):
			t.Errorf("%s: an endpoint of the id app's own, with SvelteKit's CSRF check off", path)
		case strings.HasPrefix(name, "+page.server.") && strings.Contains(read(t, path), "actions"):
			t.Errorf("%s: a form action, with SvelteKit's CSRF check off", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Each app's pages hold who is signed in, so its policy refuses what would
// let an injected tag do anything with them: no script but its own, no
// plugin, no <base> pointing its links elsewhere, no form sending what was
// typed to another site, and no frame around it.
func TestTheAppsPoliciesHoldTheirPagesIn(t *testing.T) {
	for _, app := range []string{"id", "console"} {
		config := read(t, "../../../web/"+app+"/vite.config.ts")
		for _, directive := range []string{
			`'script-src': ['self']`,
			`'object-src': ['none']`,
			`'base-uri': ['none']`,
			`'form-action': ['self']`,
			`'frame-ancestors': ['none']`,
		} {
			if !strings.Contains(config, directive) {
				t.Errorf("web/%s/vite.config.ts: the policy does not say %s", app, directive)
			}
		}
	}
}

// adapter-node no longer reads ORIGIN at run time: an app knows its own
// address only if it was built with it, and without it takes https, so the
// console refuses its own forms over plain http. Each app reads ORIGIN as it
// is built, and the stack builds each with its own address.
func TestEachAppIsBuiltWithItsOwnAddress(t *testing.T) {
	for _, app := range []string{"id", "console"} {
		if config := read(t, "../../../web/"+app+"/vite.config.ts"); !strings.Contains(config, "paths: { origin: process.env.ORIGIN") {
			t.Errorf("web/%s/vite.config.ts does not take its origin from ORIGIN as it is built", app)
		}
	}

	compose := read(t, "../../../compose.yaml")
	for app, url := range map[string]string{"web/id": "PUBLIC_URL", "web/console": "ADMIN_URL"} {
		if !strings.Contains(compose, "APP: "+app+`, VERSION: "${VERSION:-dev}", ORIGIN: "${`+url+`}"`) {
			t.Errorf("compose.yaml does not build %s with ORIGIN ${%s}", app, url)
		}
	}
	if dockerfile := read(t, "../../../web/Dockerfile"); !strings.Contains(dockerfile, `ORIGIN="${ORIGIN}" bun run build`) {
		t.Error("web/Dockerfile does not build the app with its ORIGIN")
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
