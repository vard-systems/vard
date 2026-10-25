package api

import (
	"slices"
	"strings"
	"testing"

	"vard/i18n"
	"vard/internal/api/kit/respond"
	"vard/internal/oidc"
)

// clientCodes are the `error.*` keys an app says about a request that never
// got an answer, so no server defines them: the server could not be reached,
// or answered with something that was not an answer.
var clientCodes = []string{"network", "unknown"}

// The server and the catalogs have to agree about errors in both directions.
// Every problem the server can answer with is said in the catalog of every app
// it is for — so no reader is ever shown a bare code — and says the same thing
// in English wherever it is said twice. And every `error.*` key in a catalog
// is something the server can actually answer with, so a catalog never
// carries a sentence nothing uses, and a code renamed on one side is caught.
//
// This package imports every handler, so every problem is defined by the time
// it runs.
func TestErrorCodesMatchTheCatalogs(t *testing.T) {
	defined := map[i18n.App]map[string]bool{}

	for _, problem := range respond.Problems() {
		english := ""

		for _, app := range problem.Apps {
			if defined[app] == nil {
				defined[app] = map[string]bool{}
			}
			defined[app][problem.Code] = true

			text, ok := i18n.Text(app, problem.Key())
			if !ok {
				t.Errorf("%s: %s is missing from i18n/%s/", problem.Code, problem.Key(), app)
				continue
			}

			if english != "" && text != english {
				t.Errorf("%s says %q for %s and %q for another app; one problem, one sentence",
					problem.Key(), text, app, english)
			}
			english = text
		}
	}

	for _, app := range i18n.Apps {
		for _, key := range i18n.Keys(app) {
			code, isError := strings.CutPrefix(key, "error.")
			if !isError || defined[app][code] || slices.Contains(clientCodes, code) {
				continue
			}

			t.Errorf("i18n/%s/ has %s, which the server never answers %s with", app, key, app)
		}
	}
}

// Every problem the provider can refuse something with is a problem of the
// public API, so the sign-in pages can say it.
func TestEveryProviderProblemIsAnswered(t *testing.T) {
	for _, refused := range oidc.Problems() {
		problem, ok := respond.Lookup(refused.Code)
		if !ok {
			t.Errorf("oidc.%s has no problem in the public API", refused.Code)
			continue
		}
		if !slices.Contains(problem.Apps, i18n.ID) {
			t.Errorf("%s is not for the sign-in pages", refused.Code)
		}
	}
}

// A sentence's `{parameters}` are the ones the server fills in: a translation
// that names one the server never sends shows it as it was written.
func TestErrorParametersAreSent(t *testing.T) {
	sent := map[string][]string{
		"rate_limited":                  {"seconds"},
		"account_scope_missing":         {"scope"},
		"cross_origin":                  {"origin"},
		"password_too_short":            {"min"},
		"admin_password_too_short":      {"min"},
		"language_copy_missing":         {"code"},
		"translation_too_long":          {"key", "max"},
		"sso_required":                  {"slug", "name"},
		"sso_slug_taken":                {"slug"},
		"sso_domain_taken":              {"domain"},
		"sso_domain_invalid":            {"domain"},
		"sso_discovery_failed":          {"reason"},
		"sso_metadata_invalid":          {"reason"},
		"sso_role_mapping_invalid":      {"group"},
		"mail_test_failed":              {"reason"},
		"mail_content_key_unknown":      {"key"},
		"cache_ttl_invalid":             {"max"},
		"api_scope_foreign":             {"scope"},
		"api_too_many_scopes":           {"max"},
		"application_api_scope_foreign": {"api"},
		"application_too_many_uris":     {"max"},
		"social_provider_kind_invalid":  {"kinds"},
		"user_password_too_short":       {"min"},
		"validation.max":                {"field", "max"},
		"validation.min":                {"field", "min"},
		"validation.oneof":              {"field", "values"},
		"validation.startswith":         {"field", "prefix"},
		"validation.eqfield":            {"field", "other"},
	}

	for _, problem := range respond.Problems() {
		want := sent[problem.Code]
		if strings.HasPrefix(problem.Code, "validation.") && want == nil {
			want = []string{"field"}
		}

		for _, app := range problem.Apps {
			text, _ := i18n.Text(app, problem.Key())

			for _, match := range placeholders(text) {
				if !slices.Contains(want, match) {
					t.Errorf("%s in %s names {%s}, which the server does not send", problem.Key(), app, match)
				}
			}
		}
	}
}

// placeholders are the names in a text's `{braces}`.
func placeholders(text string) []string {
	var out []string

	for {
		start := strings.IndexByte(text, '{')
		if start < 0 {
			return out
		}
		end := strings.IndexByte(text[start:], '}')
		if end < 0 {
			return out
		}

		out = append(out, text[start+1:start+end])
		text = text[start+end+1:]
	}
}
