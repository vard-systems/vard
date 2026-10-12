package apidoc

import (
	"encoding/json"
	"fmt"
	"go/constant"
	"go/types"
	"net/http"
	"slices"
	"sort"
	"strings"

	"loginer/internal/brand"
	"loginer/internal/model"
)

type page struct {
	path        string
	title       string
	description string
	order       int
	icon        string
	body        strings.Builder
}

func (p *page) line(format string, args ...any) {
	fmt.Fprintf(&p.body, format+"\n", args...)
}

func (p *page) bytes() []byte {
	var out strings.Builder
	out.WriteString("---\n")
	fmt.Fprintf(&out, "title: %s\n", quote(p.title))
	if p.description != "" {
		fmt.Fprintf(&out, "description: %s\n", quote(p.description))
	}
	fmt.Fprintf(&out, "order: %d\n", p.order)
	if p.icon != "" {
		fmt.Fprintf(&out, "icon: %s\n", p.icon)
	}
	out.WriteString("generated: true\n")
	out.WriteString("---\n\n")
	out.WriteString("<!-- Written by `make docs` from the Go source. Edit the code, not this file. -->\n\n")
	out.WriteString(strings.TrimRight(p.body.String(), "\n"))
	out.WriteString("\n")
	return []byte(out.String())
}

func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func (b *built) markdown() []*page {
	var pages []*page

	for i, server := range b.Servers {
		pages = append(pages, b.serverIndex(server, i))
		for j, section := range server.Sections {
			pages = append(pages, b.sectionPage(server, section, j+1))
		}
	}

	pages = append(pages, b.errorsPage(), b.permissionsPage())
	return pages
}

func anchor(op Operation) string {
	var out strings.Builder
	dash := false
	for _, r := range strings.ToLower(op.Method + op.Path) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
			dash = false
		} else if !dash {
			out.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimSuffix(out.String(), "-")
}

func link(server Server, section Section, op Operation) string {
	return fmt.Sprintf("/reference/%s/%s#%s", server.Key, section.Slug, anchor(op))
}

func (b *built) serverIndex(server Server, order int) *page {
	p := &page{path: server.Key + "/index.md", title: server.Title, order: order}

	if server.Key == "public" {
		p.description = "Every endpoint your application can call: the OAuth 2.0 and OpenID Connect provider, and the account API."
		p.line("| | |")
		p.line("| --- | --- |")
		p.line("| Base URL | Your issuer, such as `https://id.example.com` — the examples write it `$ISSUER` |")
		p.line("| Format | JSON, except the token, revocation and introspection endpoints, which take a form |")
		p.line("| Errors | A status and a stable `code` — see [Errors](/reference/errors) |")
		p.line("| OpenAPI | [Download](/openapi/public.json), or `GET $ISSUER/.well-known/openapi.json` |")
	} else {
		p.description = "Manage users and settings from your own code, with admin-cli, or as the panel does."
		p.line("> [!TIP]")
		p.line("> To call it from your code, use [admin-cli](/guides/admin-api): a client credentials token whose scopes are the permissions below. Signing users in to your application is the [public API](/reference/public).")
		p.line("")
		p.line("| | |")
		p.line("| --- | --- |")
		p.line("| Base URL | The panel's address — the examples write it `$ADMIN_API` |")
		p.line("| Authentication | An [admin-cli token](/guides/admin-api) (`Authorization: Bearer`), or the panel's `%s` session cookie |", brand.AdminSessionCookie)
		p.line("| Permissions | Most routes check one — see [Admin permissions](/reference/permissions) |")
		p.line("| OpenAPI | [Download](/openapi/admin.json), or `GET $ADMIN_API/api/v1/admin/openapi.json` when signed in |")
	}
	p.line("")

	for _, section := range server.Sections {
		p.line("## [%s](/reference/%s/%s)", section.Title, server.Key, section.Slug)
		p.line("")
		if section.Intro != "" {
			p.line("%s", section.Intro)
			p.line("")
		}
		p.line("| Endpoint | |")
		p.line("| --- | --- |")
		for _, op := range section.Operations {
			p.line("| [`%s %s`](%s) | %s |", op.Method, op.Path, link(server, section, op), cell(op.Summary))
		}
		p.line("")
	}

	return p
}

func (b *built) sectionPage(server Server, section Section, order int) *page {
	p := &page{
		path:        server.Key + "/" + section.Slug + ".md",
		title:       section.Title,
		description: section.Intro,
		order:       order,
	}

	for _, op := range section.Operations {
		b.operationMarkdown(p, server, section, op)
	}

	return p
}

func (b *built) operationMarkdown(p *page, server Server, section Section, op Operation) {
	p.line("## %s %s {#%s}", op.Method, op.Path, anchor(op))
	p.line("")
	if op.Summary != "" {
		p.line("%s", op.Summary)
		p.line("")
	}
	if op.Description != "" {
		p.line("%s", op.Description)
		p.line("")
	}

	p.line("| | |")
	p.line("| --- | --- |")
	p.line("| Auth | %s |", access(op))
	switch op.RateLimit {
	case "sign-in":
		p.line("| Rate limit | Per IP address |")
	case "token":
		p.line("| Rate limit | Per IP address, ten times looser than the sign-in pages |")
	}
	p.line("")

	for _, sample := range requestOf(server, op).samples() {
		p.line("```%s title=%q", sample.language, sample.title)
		p.line("%s", sample.code)
		p.line("```")
		p.line("")
	}

	var params []Param
	var places []string
	for _, group := range []struct {
		place string
		list  []Param
	}{{"path", op.PathParams}, {"query", op.QueryParams}, {"form", op.FormParams}, {"header", op.Headers}} {
		for _, param := range group.list {
			params = append(params, param)
			places = append(places, group.place)
		}
	}
	if len(params) > 0 {
		p.line("#### Parameters")
		p.line("")
		p.line("| Name | In | Description |")
		p.line("| --- | --- | --- |")
		for i, param := range params {
			name := "`" + param.Name + "`"
			if param.Required {
				name += " <small>required</small>"
			}
			p.line("| %s | %s | %s |", name, places[i], cell(param.Description))
		}
		p.line("")
	}

	if op.Body != nil {
		p.line("#### Body")
		p.line("")
		fieldTable(p, op.Body)
	}

	b.responseMarkdown(p, op)

	if len(op.Problems) > 0 {
		p.line("#### Errors")
		p.line("")
		p.line("| Status | Code | Means |")
		p.line("| --- | --- | --- |")
		for _, problem := range op.Problems {
			p.line("| %d | [`%s`](/reference/errors#%s) | %s |", problem.Status, problem.Code, problem.Code, cell(problem.English))
		}
		p.line("")
	}
}

func (b *built) responseMarkdown(p *page, op Operation) {
	if len(op.Responses) == 0 {
		return
	}

	p.line("#### Response")
	p.line("")
	for _, r := range op.Responses {
		switch {
		case r.Status == 0:
			p.line("Errors come back as an [OAuth error](/reference/errors#oauth-errors): `{\"error\", \"error_description\"}`.")
		case r.Status >= 300 && r.Status < 400:
			p.line("`%d` — redirects the browser; see the `Location` header.", r.Status)
		default:
			p.line("`%d %s`", r.Status, http.StatusText(r.Status))
		}
		p.line("")

		shaped := r.Schema != nil && (len(r.Schema.resolved().Properties) > 0 || r.Schema.Type == "array")
		if shaped && r.Status != 0 && r.ContentType == "application/json" {
			p.line("```json")
			p.line("%s", exampleJSON(r.Schema))
			p.line("```")
			p.line("")
		}
	}
}

func access(op Operation) string {
	a := op.Access
	switch a.Session {
	case "user":
		if a.Token == "account" {
			return "User session, or a user's [Account API token](/guides/account-api) with `" + accountScope(op) + "`"
		}
		return "User session — the `" + brand.UserSessionCookie + "` cookie"
	case "admin-setup":
		return "Admin session, including one still setting up a second factor"
	case "admin":
		switch {
		case a.SuperAdmin:
			return "Admin session · super admins only"
		case len(a.Permissions) > 0:
			names := make([]string, len(a.Permissions))
			for i, name := range a.Permissions {
				names[i] = "[`" + name + "`](/reference/permissions#" + strings.ReplaceAll(name, ".", "-") + ")"
			}
			who := "Admin session"
			if a.Token == "admin" {
				who = "Admin session or [admin-cli token](/guides/admin-api)"
			}
			out := who + " · needs " + strings.Join(names, " or ")
			if a.Anywhere {
				out += " for at least one application"
			}
			return out
		}
		return "Admin session"
	}

	if op.BasicAuth {
		return "Client credentials — HTTP Basic; a public client sends `client_id` alone"
	}
	for _, h := range op.Headers {
		if strings.EqualFold(h.Name, "Authorization") {
			return "Bearer access token"
		}
	}
	return "None"
}

func accountScope(op Operation) string {
	if op.Method == http.MethodGet {
		return model.ScopeAccountRead
	}
	return model.ScopeAccountWrite
}

func fieldTable(p *page, schema *Schema) {
	p.line("| Field | Type | Description |")
	p.line("| --- | --- | --- |")
	fieldRows(p, schema, "", 0)
	p.line("")
}

func fieldRows(p *page, schema *Schema, prefix string, depth int) {
	schema = schema.resolved()
	for _, property := range schema.Properties {
		name := "`" + prefix + property.Name + "`"
		if property.Required {
			name += " <small>required</small>"
		}

		description := property.Description
		if len(property.Rules) > 0 {
			description = strings.TrimSpace(description + " " + exported(strings.Join(property.Rules, ", ")) + ".")
		}
		p.line("| %s | %s | %s |", name, typeOf(property.Schema), cell(description))

		if depth >= 1 {
			continue
		}
		inner := property.Schema.resolved()
		marker := "."
		if inner.Type == "array" && inner.Items != nil {
			inner, marker = inner.Items.resolved(), "[]."
		}
		if inner.Type == "object" && len(inner.Properties) > 0 {
			fieldRows(p, inner, prefix+property.Name+marker, depth+1)
		}
	}
}

func typeOf(s *Schema) string {
	if s == nil {
		return "any"
	}

	var out string
	switch {
	case len(s.Enum) > 0:
		values := make([]string, len(s.Enum))
		for i, v := range s.Enum {
			values[i] = "`" + v + "`"
		}
		out = "one of " + strings.Join(values, ", ")
	case s.Type == "array":
		out = "array of " + typeOf(s.Items)
	case s.Type == "object" && s.Values != nil:
		out = "map of " + typeOf(s.Values)
	case s.Name != "":
		out = s.Name
	case s.Format != "":
		out = s.Type + " (" + s.Format + ")"
	case s.Type == "":
		out = "any"
	default:
		out = s.Type
	}

	if s.Nullable {
		out += " or null"
	}
	return out
}

func exampleJSON(s *Schema) string {
	out, _ := json.MarshalIndent(example(s, "", 0, false), "", "  ")
	return string(out)
}

const maxExampleDepth = 4

func example(s *Schema, name string, depth int, request bool) any {
	if s == nil {
		return nil
	}
	s = s.resolved()
	if len(s.Enum) > 0 {
		return s.Enum[0]
	}

	switch s.Type {
	case "object":
		out := orderedObject{}
		if s.Values != nil || depth >= maxExampleDepth {
			return map[string]any{}
		}
		for _, p := range s.Properties {
			if request && !p.Required && depth == 0 && len(s.Properties) > 4 {
				continue
			}
			out = append(out, field{p.Name, example(p.Schema, p.Name, depth+1, request)})
		}
		return out
	case "array":
		if s.Items.resolved() != nil && s.Items.resolved().Type == "object" && depth >= maxExampleDepth {
			return []any{}
		}
		if item := example(s.Items, strings.TrimSuffix(name, "s"), depth, request); item != nil {
			return []any{item}
		}
		return []any{}
	case "integer":
		switch {
		case strings.Contains(name, "expires_in"), strings.Contains(name, "lifetime"):
			return 3600
		case strings.HasSuffix(name, "_at"), name == "exp", name == "iat":
			return 1790000000
		}
		return 1
	case "number":
		return 1
	case "boolean":
		return !strings.HasPrefix(name, "is_disabled")
	case "string":
		return exampleString(s, name)
	}
	return map[string]any{}
}

func exampleString(s *Schema, name string) string {
	switch s.Format {
	case "date-time":
		return "2026-09-28T09:30:00Z"
	case "uuid":
		return "0199a3c2-7f10-7c1e-9b0e-3f1d2a4b5c6d"
	case "byte":
		return "aGVsbG8="
	}

	switch {
	case strings.Contains(name, "email"):
		return "ada@example.com"
	case strings.Contains(name, "redirect"), strings.Contains(name, "callback"):
		return "https://app.example.com/callback"
	case strings.Contains(name, "logo"), strings.Contains(name, "picture"), strings.Contains(name, "icon"):
		return "https://example.com/logo.png"
	case strings.HasSuffix(name, "url"), strings.HasSuffix(name, "uri"), strings.HasSuffix(name, "issuer"):
		return "https://example.com"
	case strings.Contains(name, "password"):
		return "correct horse battery staple"
	case name == "first_name", name == "given_name":
		return "Ada"
	case name == "last_name", name == "family_name":
		return "Lovelace"
	case name == "name", name == "display_name":
		return "Example"
	case name == "slug":
		return "example"
	case name == "token_type":
		return "Bearer"
	case strings.HasSuffix(name, "token"):
		return "eyJhbGciOiJSUzI1NiIs…"
	case name == "client_id":
		return "shop-web"
	case name == "scope":
		return "openid profile email"
	case name == "language", strings.HasSuffix(name, "language_code"), name == "locale":
		return "en"
	case strings.Contains(name, "phone"):
		return "+1 555 0100"
	case name == "timezone":
		return "Europe/London"
	}
	return "string"
}

type orderedObject []field

type field struct {
	name  string
	value any
}

func (o orderedObject) MarshalJSON() ([]byte, error) {
	var out strings.Builder
	out.WriteByte('{')
	for i, f := range o {
		if i > 0 {
			out.WriteByte(',')
		}
		key, _ := json.Marshal(f.name)
		value, err := json.Marshal(f.value)
		if err != nil {
			return nil, err
		}
		out.Write(key)
		out.WriteByte(':')
		out.Write(value)
	}
	out.WriteByte('}')
	return []byte(out.String()), nil
}

func (b *built) errorsPage() *page {
	p := &page{
		path:        "errors.md",
		icon:        "error-warning",
		title:       "Errors",
		description: "Every error code either server answers with, and what it means.",
		order:       10,
	}

	p.line("Both servers answer an error the same way: a status, and a body with a stable `code` to act on, the English `error` to show a developer, and the `params` that fill in its `{placeholders}`.")
	p.line("")
	p.line("```json")
	p.line(`{ "error": "Too many attempts. Try again in 30 seconds.", "code": "rate_limited", "params": { "seconds": 30 } }`)
	p.line("```")
	p.line("")
	p.line("Match on `code`, never on `error`: the sentence may be reworded, and the sign-in pages show it in the reader's language. Any endpoint can also answer `500` with `internal` when the server itself fails; the cause is in its log, never in the answer.")
	p.line("")
	p.line("The provider's own endpoints — token, userinfo, revocation, introspection — are the exception: they follow their RFCs and answer [OAuth errors](#oauth-errors).")
	p.line("")

	groups := []struct{ title, app, id string }{
		{"Public API", "id", "public-api"},
		{"Admin API", "console", "admin-api"},
	}
	for _, group := range groups {
		p.line("## %s {#%s}", group.title, group.id)
		p.line("")
		p.line("| Status | Code | Means |")
		p.line("| --- | --- | --- |")
		for _, problem := range b.Problems {
			if !slices.Contains(problem.Apps, group.app) {
				continue
			}
			p.line("| %d | <span id=\"%s\"></span>`%s` | %s |", problem.Status, problem.Code, problem.Code, cell(problem.English))
		}
		p.line("")
	}

	p.line("## OAuth errors {#oauth-errors}")
	p.line("")
	p.line("The token, userinfo, revocation and introspection endpoints answer `{\"error\": \"<code>\", \"error_description\": \"<sentence>\"}` (RFC 6749 section 5.2). The authorization endpoint sends the same two back to the application's `redirect_uri`, with its `state` — once the redirect URI is known to be the application's. Before that, the browser is shown an error page instead, so an unregistered address never receives anything.")
	p.line("")
	p.line("| Code | Status |")
	p.line("| --- | --- |")
	for _, code := range b.oauthCodes() {
		status := "400"
		switch code {
		case "invalid_client", "invalid_token":
			status = "401"
		case "insufficient_scope":
			status = "403"
		case "server_error":
			status = "500"
		case "login_required", "access_denied", "unsupported_response_type":
			status = "redirect"
		}
		p.line("| `%s` | %s |", code, status)
	}
	p.line("")

	return p
}

func (b *built) oauthCodes() []string {
	pkg, err := b.loader.suffixed("/internal/oidc")
	if err != nil {
		return nil
	}
	var out []string
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		c, ok := scope.Lookup(name).(*types.Const)
		if !ok || !strings.HasPrefix(name, "Err") || c.Val().Kind() != constant.String {
			continue
		}
		out = append(out, constant.StringVal(c.Val()))
	}
	sort.Strings(out)
	return out
}

func (b *built) permissionsPage() *page {
	p := &page{
		path:        "permissions.md",
		icon:        "shield-keyhole",
		title:       "Admin permissions",
		description: "What each permission an admin role can grant allows, and the routes that check it.",
		order:       11,
	}

	p.line("An administrator's roles grant permissions from this list. A role assigned for one application grants its *scopable* permissions for that application only. Managing administrators, their roles, mail, one-time codes, signing keys and the cache is not here: those belong to the super admin role alone.")
	p.line("")

	var admin Server
	for _, server := range b.Servers {
		if server.Key == "admin" {
			admin = server
		}
	}

	group := ""
	for _, permission := range model.AdminPermissions {
		if permission.Group != group {
			group = permission.Group
			p.line("## %s", group)
			p.line("")
		}

		p.line("### %s {#%s}", permission.Name, strings.ReplaceAll(permission.Name, ".", "-"))
		p.line("")
		p.line("%s.", strings.TrimSuffix(permission.Description, "."))
		if permission.Scopable {
			p.line("Can be granted for a single application.")
		}
		p.line("")

		var rows []string
		for _, section := range admin.Sections {
			for _, op := range section.Operations {
				if slices.Contains(op.Access.Permissions, permission.Name) {
					rows = append(rows, fmt.Sprintf("- [`%s %s`](%s) — %s", op.Method, op.Path, link(admin, section, op), op.Summary))
				}
			}
		}
		if len(rows) > 0 {
			p.line("Checked by:")
			p.line("")
			for _, row := range rows {
				p.line("%s", row)
			}
			p.line("")
		}
	}

	return p
}

func cell(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	return strings.ReplaceAll(text, "|", "\\|")
}
