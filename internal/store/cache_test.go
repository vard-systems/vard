package store

import (
	"reflect"
	"strings"
	"testing"

	"loginer/internal/model"
)

// What the store caches goes to Redis as JSON, and a field JSON leaves out
// comes back empty. For most fields that would be a quiet bug — a cached
// flow without its steps — and for a secret it would be worse: a copy that
// looks whole and is not. So every field of every cached type has to be one
// JSON writes, and the only exceptions are named here with why.
func TestCachedTypesSurviveJSON(t *testing.T) {
	// Fields JSON may leave out of a cached row.
	allowed := map[string]string{
		"Language.Translations": "a relation the cached reads never load",

		// An administrator is kept as a principal, which spells the roles and
		// factors out and puts them back; the rest is what a request never
		// needs, and what AdminPrincipal says not to use it for.
		"Admin.Assignments":            "kept beside it in principal, and put back",
		"Admin.MFA":                    "kept beside it in principal, and put back",
		"Admin.PasswordHash":           "never kept: a password is checked against AdminByID",
		"Admin.FailedLoginCount":       "counted in the database; only locked_until decides a sign-in",
		"Admin.Sessions":               "a relation AdminByID never loads",
		"Admin.Service":                "only model.ServiceAdmin sets it, per request, and that is never stored or cached",
		"AdminRoleAssignment.Admin":    "the administrator it belongs to, who is the principal",
		"MFA.Admin":                    "the administrator it belongs to, who is the principal",
		"MFA.Secret":                   "never kept: a code is checked against the factor in the database",
		"MFA.RecoveryCodes":            "never kept: a recovery code is spent in the database",
		"AdminSession.Admin":           "a relation the session read never loads",
		"AdminSession.TokenHash":       "the key it is kept under, put back when it is read",
		"Application.ClientSecretHash": "kept beside it in client, and put back",
		"SSOConnection.ClientSecret":   "never kept: only the sign-in buttons are cached, which read the name",
		"SSOConnection.SPKey":          "never kept: only the sign-in buttons are cached, which read the name",
		"UserRole.Application":         "a relation the role graph never loads",
	}

	cachedTypes := []any{
		model.Language{},
		model.Organization{},
		model.LoginFlow{},
		SocialButton{},
		model.SSOConnection{},
		model.OTPSettings{},
		model.AdminSecurity{},
		model.AdminSession{},
		userSession{},
		principal{},
		model.Admin{},
		model.AdminRoleAssignment{},
		model.MFA{},
		client{},
		model.Application{},
		model.UserRole{},
		model.APIScope{},
		model.API{},
		Audience{},
	}

	for _, value := range cachedTypes {
		typ := reflect.TypeOf(value)

		t.Run(typ.Name(), func(t *testing.T) {
			for _, path := range fieldsJSONDrops(typ, typ.Name()) {
				if _, ok := allowed[path]; !ok {
					t.Errorf("%s is tagged json:\"-\", so a cached copy would lose it; cache something without it, or say here why that is fine", path)
				}
			}
		})
	}
}

// fieldsJSONDrops names every exported field, embedded ones included, that
// encoding/json would not write.
func fieldsJSONDrops(typ reflect.Type, name string) []string {
	var out []string

	for field := range typ.Fields() {
		if !field.IsExported() {
			continue
		}

		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			out = append(out, fieldsJSONDrops(field.Type, field.Type.Name())...)
			continue
		}

		if tag, _, _ := strings.Cut(field.Tag.Get("json"), ","); tag == "-" {
			out = append(out, name+"."+field.Name)
		}
	}

	return out
}
