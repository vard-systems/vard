package apidoc

import (
	"net/http"
	"sort"
)

var servers = []struct {
	key, title, function string
}{
	{"public", "Public API", "registerPublicRoutes"},
	{"admin", "Admin API", "registerAdminRoutes"},
}

var sections = map[string]struct{ title, summary string }{
	"api":          {"Health", "Whether the server is up, for load balancers and uptime checks."},
	"reference":    {"OpenAPI document", "The server's own OpenAPI document, for your tooling."},
	"oauth":        {"OAuth 2.0 and OpenID Connect", "The endpoints your application signs users in with, and gets and checks tokens from."},
	"account":      {"Account", "The API behind the hosted sign-in pages and a user's account page. Applications do not usually call it."},
	"setup":        {"First administrator", "Creating the first administrator of a new installation."},
	"auth":         {"Signing in", "Signing an administrator in and out, and who is signed in."},
	"mfa":          {"Second factor", "An administrator's authenticator app and recovery codes."},
	"users":        {"Users", "The people your organisation manages."},
	"sessions":     {"User sessions", "Who is signed in, and signing them out."},
	"fields":       {"User fields", "The custom fields a user record is made of."},
	"roles":        {"User roles", "The roles users hold, globally and in each application."},
	"admins":       {"Administrators", "The panel's administrators."},
	"adminroles":   {"Administrator roles", "What administrators may do in the panel."},
	"applications": {"Applications", "The applications that sign users in: their settings, secrets and API access."},
	"apis":         {"APIs", "The APIs access tokens are issued for, and their scopes."},
	"organization": {"Organization", "The organisation's name, branding and agreements."},
	"social":       {"Social providers", "Signing in with an account at another provider."},
	"sso":          {"Single sign-on", "Signing in through a customer's own identity provider, over OpenID Connect or SAML."},
	"flows":        {"Login flows", "The steps a sign-in goes through, per application."},
	"languages":    {"Languages", "The sign-in pages' languages and their text."},
	"mail":         {"Mail", "How email is sent, and what each message says."},
	"otp":          {"One-time codes", "How the one-time codes sent by email behave."},
	"activity":     {"Activity", "The dashboard's counts and the activity log."},
	"keys":         {"Signing keys", "The keys tokens are signed with, and rotating them."},
	"caching":      {"Cache", "What the Redis cache holds, and clearing it."},
}

type built struct {
	Reference
	schemas map[string]*schemas
	loader  *loader
}

func build(root string) (*built, error) {
	l, err := load(root)
	if err != nil {
		return nil, err
	}

	out := &built{schemas: map[string]*schemas{}, loader: l}
	out.Problems = l.Problems()

	for _, definition := range servers {
		routes, err := l.routes(definition.function)
		if err != nil {
			return nil, err
		}

		s := newSchemas(l)
		out.schemas[definition.key] = s
		server := Server{Key: definition.key, Title: definition.title}

		index := map[string]int{}
		for _, r := range routes {
			pkg := r.handler.Pkg()
			slug := pkg.Name()

			i, ok := index[slug]
			if !ok {
				section := Section{Slug: slug, Title: exported(slug)}
				if known, ok := sections[slug]; ok {
					section.Title, section.Intro = known.title, known.summary
				}
				if slug == "api" {
					section.Slug = "health"
				}
				i = len(server.Sections)
				index[slug] = i
				server.Sections = append(server.Sections, section)
			}

			op := l.operation(r, s)
			l.guardProblems(&op)
			server.Sections[i].Operations = append(server.Sections[i].Operations, op)
		}

		out.Servers = append(out.Servers, server)
	}

	return out, nil
}

func (l *loader) guardProblems(op *Operation) {
	var codes []string
	if op.Access.Session != "" {
		codes = append(codes, "not_signed_in")
	}
	if len(op.Access.Permissions) > 0 || op.Access.SuperAdmin {
		codes = append(codes, "forbidden")
	}
	if op.RateLimit != "" {
		codes = append(codes, "rate_limited")
	}
	if op.OriginChecked {
		codes = append(codes, "cross_origin")
	}
	switch op.Access.Token {
	case "admin":
		codes = append(codes, "token_refused")
	case "account":
		codes = append(codes, "account_token_refused", "account_scope_missing")
	}

	have := map[string]bool{}
	for _, problem := range op.Problems {
		have[problem.Code] = true
	}

	for _, code := range codes {
		if have[code] {
			continue
		}
		for _, problem := range l.problems {
			if problem.Code == code && problem.Status < http.StatusInternalServerError {
				op.Problems = append(op.Problems, problem)
				break
			}
		}
	}

	sort.Slice(op.Problems, func(i, j int) bool {
		a, b := op.Problems[i], op.Problems[j]
		if a.Status != b.Status {
			return a.Status < b.Status
		}
		return a.Code < b.Code
	})
}
