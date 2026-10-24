package model

import (
	"slices"
	"sort"
	"strings"
	"time"
)

type TokenRequest struct {
	Application Application

	User  *User
	Roles []UserRole

	API        *API
	Authorized bool
	Allowed    []string

	Requested []string

	Issuer string
	Now    time.Time
}

// ScopeDecision is what happened to one requested scope, and why.
type ScopeDecision struct {
	Scope   string `json:"scope"`
	Kind    string `json:"kind"` // "openid" or "api"
	Granted bool   `json:"granted"`
	Reason  string `json:"reason"`
	// IsDefault marks an API scope that was not asked for, but added because
	// the API makes it a default.
	IsDefault bool `json:"is_default,omitempty"`
}

// TokenPreview is what a token request amounts to: whether a token is issued
// at all, the decision on every scope, and the claims each token carries.
type TokenPreview struct {
	Issued    bool            `json:"issued"`
	Reason    string          `json:"reason,omitempty"`
	Decisions []ScopeDecision `json:"decisions"`

	// AccessTokenHeader is the JOSE header of the access token: how it is
	// signed, and that it is a JWT access token (RFC 9068).
	AccessTokenHeader map[string]any `json:"access_token_header,omitempty"`
	AccessToken       map[string]any `json:"access_token,omitempty"`
	IDToken           map[string]any `json:"id_token,omitempty"`
}

const (
	ClaimGlobalRoles      = "global_roles"
	ClaimApplicationRoles = "roles"
)

// EvaluateToken decides what a token request amounts to; it is the one place
// down-scoping happens:
//
//   - no token for a disabled application, a missing grant, an inactive user,
//     or a user without a required role;
//   - no token for an API the application is not authorised for;
//   - an OIDC scope is granted when the application may use it;
//   - an API scope (and the audience's defaults) is granted when the
//     application is allowed it and, when the API enforces roles, the user's
//     roles grant it;
//   - offline_access is refused for an API that does not allow refresh tokens;
//   - the audience is the requested API alone, and roles are carried only when
//     the application asserts them and "roles" is granted.
func EvaluateToken(r TokenRequest) TokenPreview {
	app := r.Application
	preview := TokenPreview{Decisions: []ScopeDecision{}}

	deny := func(reason string) TokenPreview {
		preview.Reason = reason
		return preview
	}

	switch {
	case !app.IsEnabled:
		return deny("the application is disabled")
	case r.User == nil && !slices.Contains(app.GrantTypes, GrantClientCredentials):
		return deny("a token without a user needs the client_credentials grant, which the application does not have")
	case r.User != nil && !app.SignsUsersIn():
		return deny("a token for a user needs the authorization_code or device code grant, which the application does not have")
	case r.User != nil && !r.User.IsActive:
		return deny("the user is inactive")
	case r.API != nil && !r.Authorized:
		return deny("the application is not authorised for " + r.API.Identifier)
	}

	appRoles := []string{}
	globalRoles := []string{}
	granting := map[string][]string{}

	for _, role := range r.Roles {
		switch {
		case role.Global():
			globalRoles = append(globalRoles, role.Name)
		case role.In(&app.ID):
			appRoles = append(appRoles, role.Name)
		default:
			// Another application's roles grant nothing here, or its managers
			// could add scopes to this application's tokens.
			continue
		}

		for _, scope := range role.APIScopes {
			granting[scope.ID.String()] = append(granting[scope.ID.String()], role.Name)
		}
	}
	sort.Strings(appRoles)
	sort.Strings(globalRoles)

	if r.User != nil && app.RequireRoleAssignment && len(appRoles) == 0 {
		return deny("the application requires a role, and the user holds none of its roles")
	}

	granted := []string{}
	decide := func(scope, kind string, ok bool, reason string) {
		preview.Decisions = append(preview.Decisions, ScopeDecision{Scope: scope, Kind: kind, Granted: ok, Reason: reason})
		if ok {
			granted = append(granted, scope)
		}
	}

	apiDecision := func(apiScope APIScope) (bool, string) {
		switch {
		case !slices.Contains(r.Allowed, apiScope.Name):
			return false, "the application is not allowed this scope"
		case r.User == nil:
			return true, "the application is allowed this scope"
		case !r.API.EnforceRoles:
			return true, "the application is allowed it, and the API does not enforce roles"
		case len(granting[apiScope.ID.String()]) == 0:
			return false, "none of the user's roles grant this scope"
		default:
			return true, "granted by " + strings.Join(unique(granting[apiScope.ID.String()]), ", ")
		}
	}

	requested := unique(r.Requested)

	for _, scope := range requested {
		if slices.Contains(Scopes, scope) {
			switch {
			case r.User == nil:
				decide(scope, "openid", false, "there is no user to describe")
			case !slices.Contains(app.Scopes, scope):
				decide(scope, "openid", false, "the application may not use this scope")
			case scope == ScopeOfflineAccess && !slices.Contains(app.GrantTypes, GrantRefreshToken):
				decide(scope, "openid", false, "a refresh token needs the refresh_token grant")
			case scope == ScopeOfflineAccess && r.API != nil && !r.API.AllowOfflineAccess:
				decide(scope, "openid", false, r.API.Identifier+" does not allow refresh tokens")
			default:
				decide(scope, "openid", true, "the application may use this scope")
			}
			continue
		}

		apiScope, known := findScope(r.API, scope)
		switch {
		case r.API == nil:
			decide(scope, "api", false, "no audience was requested, so API scopes cannot be granted")
		case !known:
			decide(scope, "api", false, r.API.Identifier+" has no such scope")
		default:
			ok, reason := apiDecision(apiScope)
			decide(scope, "api", ok, reason)
		}
	}

	if r.API != nil {
		for _, apiScope := range r.API.Scopes {
			if !apiScope.IsDefault || slices.Contains(requested, apiScope.Name) {
				continue
			}

			ok, reason := apiDecision(apiScope)
			preview.Decisions = append(preview.Decisions, ScopeDecision{
				Scope:     apiScope.Name,
				Kind:      "api",
				Granted:   ok,
				Reason:    "default scope: " + reason,
				IsDefault: true,
			})
			if ok {
				granted = append(granted, apiScope.Name)
			}
		}
	}

	preview.Issued = true

	subject := app.ClientID
	if r.User != nil {
		subject = r.User.ID.String()
	}

	audience := []string{}
	if r.API != nil {
		audience = append(audience, r.API.Identifier)
	}

	lifetime := app.AccessTokenLifetime
	algorithm := AlgRS256
	if r.API != nil {
		if r.API.TokenLifetime > 0 {
			lifetime = r.API.TokenLifetime
		}
		if r.API.SigningAlgorithm != "" {
			algorithm = r.API.SigningAlgorithm
		}
	}

	preview.AccessTokenHeader = map[string]any{"alg": algorithm, "typ": "at+jwt"}

	access := map[string]any{
		"iss":       r.Issuer,
		"sub":       subject,
		"aud":       audience,
		"azp":       app.ClientID,
		"client_id": app.ClientID,
		"iat":       r.Now.Unix(),
		"exp":       r.Now.Add(time.Duration(lifetime) * time.Second).Unix(),
		"scope":     strings.Join(granted, " "),
	}

	rolesGranted := slices.Contains(granted, ScopeRoles)
	if r.User != nil && app.AssertRoles && rolesGranted {
		access[ClaimApplicationRoles] = appRoles
		access[ClaimGlobalRoles] = globalRoles
	}
	preview.AccessToken = access

	if r.User != nil && slices.Contains(granted, ScopeOpenID) {
		id := map[string]any{
			"iss": r.Issuer,
			"sub": r.User.ID.String(),
			"aud": app.ClientID,
			"iat": r.Now.Unix(),
			"exp": r.Now.Add(time.Duration(app.IDTokenLifetime) * time.Second).Unix(),
		}

		if slices.Contains(granted, ScopeProfile) {
			id["name"] = strings.TrimSpace(r.User.FirstName + " " + r.User.LastName)
			id["given_name"] = r.User.FirstName
			id["family_name"] = r.User.LastName
		}
		if slices.Contains(granted, ScopeEmail) {
			id["email"] = r.User.Email
			id["email_verified"] = r.User.IsEmailVerified
		}
		if app.AssertRoles && rolesGranted {
			id[ClaimApplicationRoles] = appRoles
			id[ClaimGlobalRoles] = globalRoles
		}

		preview.IDToken = id
	}

	return preview
}

func findScope(api *API, name string) (APIScope, bool) {
	if api == nil {
		return APIScope{}, false
	}

	for _, scope := range api.Scopes {
		if scope.Name == name {
			return scope, true
		}
	}

	return APIScope{}, false
}

func unique(values []string) []string {
	out := []string{}
	for _, value := range values {
		if value != "" && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}

	return out
}
