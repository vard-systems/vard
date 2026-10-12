package apidoc

import (
	"strconv"
	"strings"
	"time"

	"loginer/internal/model"
)

var protocolParams = map[string]string{
	"client_id":             "The application's client ID.",
	"client_secret":         "The application's secret, when it sends it in the form rather than with HTTP Basic. Never sent by a public client.",
	"grant_type":            "One of " + codes(model.GrantTypes) + ".",
	"code":                  "The authorization code the browser was sent back with. It can be spent once, within " + minutes(model.AuthorizationCodeLifetime) + ".",
	"redirect_uri":          "Where the browser is sent back to. It has to match one of the application's registered redirect URIs exactly; at the token endpoint, it is the one the authorization request named.",
	"code_verifier":         "The PKCE verifier whose S256 hash was sent as `code_challenge`.",
	"refresh_token":         "The refresh token to exchange. Refresh tokens rotate: the one sent is spent, and the answer carries the next.",
	"scope":                 "Space-separated scopes: " + codes(model.Scopes) + ", and the scopes of the API named in `audience`.",
	"audience":              "The identifier of the API the access token is for, as it is registered on the panel's APIs page.",
	"response_type":         "Always `code`.",
	"response_mode":         "`query`, or left out.",
	"state":                 "An opaque value sent back unchanged with the answer, which ties the answer to the request that asked for it.",
	"nonce":                 "A value copied into the ID token, which ties the token to this sign-in.",
	"code_challenge":        "The base64url SHA-256 of the code verifier, 43 characters. Required when the application requires PKCE, as every public client does.",
	"code_challenge_method": "`S256`, the only method accepted.",
	"prompt":                "`none` to be sent back with `login_required` rather than shown a sign-in page, or `login` to ask for the password even when the user is signed in.",
	"max_age":               "Seconds since the user last entered their password, past which they are asked again.",
	"login_hint":            "An email address to fill in on the sign-in page.",
	"id_token_hint":         "An ID token this server issued, which says whose session to end. It may have expired.",
	"post_logout_redirect_uri": "Where to send the browser once signed out. It has to be one the application registered; without one, the browser " +
		"lands on the sign-in app's signed-out page.",
	"token":        "The token the request is about.",
	"access_token": "The access token, for a client that cannot send an Authorization header.",
}

func minutes(d time.Duration) string {
	if n := int(d.Minutes()); n != 1 {
		return strconv.Itoa(n) + " minutes"
	}
	return "a minute"
}

func codes(values []string) string {
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "`" + v + "`"
	}
	return strings.Join(quoted, ", ")
}
