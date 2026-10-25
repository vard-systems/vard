// Package brand is the server's own record of what the project calls itself,
// and every name built from it: cookies, the Redis prefix, locks, the API
// identifiers.
//
// Each app keeps its own brand in src/lib/brand.ts, which the server never
// reads. They may name the product and pick a favicon as they like; what has
// to agree, the slug and the cookie names the server reads back, brand_test.go
// holds together.
//
// What changes with a deployment (addresses, mail sender) is configuration,
// in internal/config. `make rename` renames the project everywhere.
package brand

import (
	_ "embed"
	"encoding/base64"
)

const (
	// Name is what a person reads: logs, mail, authenticator apps, passkey
	// prompts.
	Name = "Vard"

	// Slug is what identifiers are built from: lower-case letters and digits,
	// so it is a valid cookie, Redis, URN, Postgres and environment name.
	Slug = "vard"

	EnvPrefix = "VARD_"
)

const (
	AdminSessionCookie = Slug + "_session"

	// UserSessionCookie carries a person's own session for single sign-on
	// between applications. It is deliberately not the administrators' cookie:
	// signing in to an application must never sign anyone in to the panel.
	UserSessionCookie = Slug + "_user_session"

	SignInStateCookie = Slug + "_sign_in_state"

	// SecondFactorCookie names a sign-in waiting for the user's second
	// factor, so only the browser that signed in can finish it.
	SecondFactorCookie = Slug + "_second_factor"

	LanguageCookie = Slug + "-account-language"
)

const (
	AdminAPIIdentifier   = "urn:" + Slug + ":admin-api"
	AccountAPIIdentifier = "urn:" + Slug + ":account-api"
)

const RedisPrefix = Slug + ":"

const Realm = Name

const (
	// AdminLock is a number, not a name, so a rename leaves it as it is.
	AdminLock int64 = 0x6c6f67696e6572

	// SigningKeysLock is named rather than numbered, because this one is worth
	// being able to read in pg_locks.
	SigningKeysLock = Slug + "_signing_keys"
)

// favicon is the server's own, for the pages it renders itself.
//
//go:embed favicon.svg
var favicon string

// FaviconURL is the favicon as a data: URL: the server's pages have no asset
// of their own to point at.
var FaviconURL = "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(favicon))
