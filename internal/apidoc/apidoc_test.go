package apidoc

import (
	"errors"
	"go/ast"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"

	"loginer/internal/api"
	"loginer/internal/config"
)

// The reference is committed, so the docs app and the server can be built
// without running the generator. This is what keeps it honest: a route, a
// doc comment, a request field or a problem changed without `make docs`
// fails here, the way the panel's tokens fail `bun run check`.
func TestTheReferenceIsCurrent(t *testing.T) {
	root := repoRoot(t)

	files, err := Generate(root)
	if err != nil {
		t.Fatal(err)
	}

	stale, err := Stale(root, files)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range stale {
		t.Errorf("%s is behind the code; run make docs", path)
	}
}

// The reference is read from the source of the route table, and Gin builds
// the servers from the same function at run time. Reading source can miss
// what running it does not — a route mounted in a loop, say — so the two are
// held to each other: every route either server really answers is in the
// reference, and the reference names nothing they do not.
func TestEveryMountedRouteIsInTheReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	public, err := api.NewPublic(config.Config{AccountURL: "http://localhost"}, log, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	admin, err := api.NewAdmin(config.Config{AdminURL: "http://localhost"}, nil, log, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	b := reference(t)

	engines := map[string]*gin.Engine{"public": public, "admin": admin}
	for _, server := range b.Servers {
		t.Run(server.Key, func(t *testing.T) {
			var mounted, documented []string
			for _, route := range engines[server.Key].Routes() {
				mounted = append(mounted, route.Method+" "+route.Path)
			}
			for _, op := range server.Operations() {
				documented = append(documented, op.Method+" "+op.Path)
			}

			for _, missing := range difference(mounted, documented) {
				t.Errorf("%s is mounted but not in the reference", missing)
			}
			for _, extra := range difference(documented, mounted) {
				t.Errorf("%s is in the reference but not mounted", extra)
			}
		})
	}
}

// What the reference reads out of the route table's guards, for routes whose
// guards are worth pinning down: if one of these changes, either the table
// or the reading of it changed, and a reader would be told the wrong thing.
func TestGuardsAreReadFromTheRouteTable(t *testing.T) {
	b := reference(t)

	find := func(method, path string) Operation {
		for _, server := range b.Servers {
			for _, op := range server.Operations() {
				if op.Method == method && op.Path == path {
					return op
				}
			}
		}
		t.Fatalf("%s %s is not in the reference", method, path)
		return Operation{}
	}

	tests := []struct {
		name          string
		method, path  string
		session       string
		token         string
		permissions   []string
		anywhere      bool
		superAdmin    bool
		rateLimit     string
		originChecked bool
	}{
		{name: "the token endpoint takes no session and has its own limit",
			method: "POST", path: "/oauth2/token", rateLimit: "token"},
		{name: "signing in is limited and checks the origin",
			method: "POST", path: "/api/v1/account/login", rateLimit: "sign-in", originChecked: true},
		{name: "a user's own profile takes their session or an account API token",
			method: "GET", path: "/api/v1/account/me", session: "user", token: "account"},
		{name: "listing users takes users.read, from a session or an admin-cli token",
			method: "GET", path: "/api/v1/admin/users", session: "admin", token: "admin", permissions: []string{"users.read"}},
		{name: "applications are reachable with the permission for one of them",
			method: "GET", path: "/api/v1/admin/applications", session: "admin", token: "admin", permissions: []string{"applications.read"}, anywhere: true},
		{name: "an administrator's own account takes their session alone",
			method: "GET", path: "/api/v1/admin/me", session: "admin"},
		{name: "rotating signing keys is a super admin's alone, and limited",
			method: "POST", path: "/api/v1/admin/signing-keys/rotate", session: "admin", superAdmin: true, rateLimit: "sign-in", originChecked: true},
		{name: "setting up an authenticator takes a session part way through",
			method: "GET", path: "/api/v1/admin/mfa", session: "admin-setup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := find(tt.method, tt.path)
			a := op.Access
			if a.Session != tt.session || a.Token != tt.token || strings.Join(a.Permissions, ",") != strings.Join(tt.permissions, ",") ||
				a.Anywhere != tt.anywhere || a.SuperAdmin != tt.superAdmin {
				t.Errorf("access = %+v", a)
			}
			if op.RateLimit != tt.rateLimit {
				t.Errorf("rate limit = %q, want %q", op.RateLimit, tt.rateLimit)
			}
			if op.OriginChecked != tt.originChecked {
				t.Errorf("origin checked = %v, want %v", op.OriginChecked, tt.originChecked)
			}
		})
	}
}

// A handler's request type, the problems it names, and its doc comment all
// reach the reference.
func TestHandlersAreReadIntoOperations(t *testing.T) {
	b := reference(t)

	var login Operation
	for _, op := range b.Servers[0].Operations() {
		if op.Method == "POST" && op.Path == "/api/v1/account/login" {
			login = op
		}
	}

	if login.Summary == "" {
		t.Error("the handler's doc comment was not read")
	}
	if login.Body == nil {
		t.Fatal("the request body was not read")
	}

	fields := map[string]Property{}
	for _, property := range login.Body.Properties {
		fields[property.Name] = property
	}
	if !fields["email"].Required || !fields["password"].Required {
		t.Errorf("email and password are required by the validate tags: %+v", fields)
	}
	if fields["remember"].Required {
		t.Error("remember is not required")
	}

	codes := map[string]bool{}
	for _, problem := range login.Problems {
		codes[problem.Code] = true
	}
	for _, code := range []string{"invalid_body", "rate_limited"} {
		if !codes[code] {
			t.Errorf("POST /api/v1/account/login does not list %s", code)
		}
	}
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name, method, doc, summary, rest string
	}{
		{"the Go name is taken off", "Login", "Login signs a user in.", "Signs a user in.", ""},
		{"the rest is the description", "List", "List returns a page.\n\nNewest first.", "Returns a page.", "Newest first."},
		{"an abbreviation does not end the sentence", "Get", "Get finds one, e.g. by id. Then more.", "Finds one, e.g. by id.", "Then more."},
		{"a comment that is not about the name is kept", "Token", "Issues tokens.", "Issues tokens.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary, rest := summary(tt.method, comment(tt.doc))
			if summary != tt.summary || rest != tt.rest {
				t.Errorf("summary(%q) = %q, %q; want %q, %q", tt.doc, summary, rest, tt.summary, tt.rest)
			}
		})
	}
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name, tag, kind string
		rules           []string
		required        bool
		enum            []string
	}{
		{name: "a required email", tag: "required,email,max=255", kind: "string",
			rules: []string{"an email address", "at most 255 characters"}, required: true},
		{name: "a list's size is in items", tag: "max=20", kind: "array", rules: []string{"at most 20 items"}},
		{name: "a number has no unit", tag: "min=60", kind: "integer", rules: []string{"at least 60"}},
		{name: "oneof is an enum", tag: "oneof=web spa", kind: "string", enum: []string{"web", "spa"}},
		{name: "no tag, no rules", tag: "", kind: "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := &Schema{Type: tt.kind}
			rules, required := validation(tt.tag, schema)
			if strings.Join(rules, "|") != strings.Join(tt.rules, "|") || required != tt.required {
				t.Errorf("validation(%q) = %q, %v", tt.tag, rules, required)
			}
			if strings.Join(schema.Enum, ",") != strings.Join(tt.enum, ",") {
				t.Errorf("enum = %q, want %q", schema.Enum, tt.enum)
			}
		})
	}
}

// A field's comment and rules are read by callers, who send the JSON names.
func TestFieldsAreNamedAsTheyAreSent(t *testing.T) {
	wire := map[string]string{"Password": "password", "IsPasswordTemporary": "is_password_temporary", "IsActive": "is_active"}

	tests := []struct{ name, text, want string }{
		{"a comment naming two fields", "IsActive and Password may be left out.", "`is_active` and `password` may be left out."},
		{"the longer name wins", "IsPasswordTemporary marks it.", "`is_password_temporary` marks it."},
		{"a word inside another is left", "Passwords are hashed.", "Passwords are hashed."},
		{"an eqfield rule", "the same as Password", "the same as `password`"},
	}

	rename := wireNames(wire)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rename(tt.text); got != tt.want {
				t.Errorf("wireNames(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}

func TestPaths(t *testing.T) {
	tests := []struct{ gin, openAPI, anchor string }{
		{"/api/v1/admin/users/:id", "/api/v1/admin/users/{id}", "get-api-v1-admin-users-id"},
		{"/.well-known/openid-configuration", "/.well-known/openid-configuration", "get-well-known-openid-configuration"},
		{"/oauth2/token", "/oauth2/token", "get-oauth2-token"},
	}

	for _, tt := range tests {
		t.Run(tt.gin, func(t *testing.T) {
			if got := openAPIPath(tt.gin); got != tt.openAPI {
				t.Errorf("openAPIPath = %q, want %q", got, tt.openAPI)
			}
			if got := anchor(Operation{Method: "GET", Path: tt.gin}); got != tt.anchor {
				t.Errorf("anchor = %q, want %q", got, tt.anchor)
			}
		})
	}
}

// comment is a doc comment reading text, as the parser would make it.
func comment(text string) *ast.CommentGroup {
	group := &ast.CommentGroup{}
	for line := range strings.SplitSeq(text, "\n") {
		group.List = append(group.List, &ast.Comment{Text: strings.TrimRight("// "+line, " ")})
	}
	return group
}

func difference(a, b []string) []string {
	in := map[string]bool{}
	for _, v := range b {
		in[v] = true
	}
	var out []string
	for _, v := range a {
		if !in[v] {
			out = append(out, v)
		}
	}
	sort.Strings(out)
	return out
}

// loaded is the module read once for every test that needs the reference:
// type-checking the source is the slow part of each.
var loaded = sync.OnceValues(func() (*built, error) {
	root, err := findRoot()
	if err != nil {
		return nil, err
	}
	return build(root)
})

func reference(t *testing.T) *built {
	t.Helper()
	b, err := loaded()
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// repoRoot is the directory go.mod is in.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the test")
		}
		dir = parent
	}
}
