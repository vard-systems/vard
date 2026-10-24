// Package brand is what this project calls itself, and every name built from
// it: cookies, the Redis prefix, locks. Each app keeps the same list in its
// brand.ts, and brand_test.go checks the two sides agree.
//
// Renaming the project is these constants, the module path in go.mod, each
// app's brand.ts, and a `git mv` of cmd/<name>. A test fails on the name
// spelled anywhere else.
package brand

const (
	Name = "Loginer"

	Slug = "loginer"

	EnvPrefix = "LOGINER_"

	GitHubURL = "https://github.com/loginer/loginer"
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

// FlowSchema is the $schema of exported login flows; changing it orphans every
// exported file.
const FlowSchema = Slug + ".login-flow/1"

const (
	AdminLock int64 = 0x6c6f67696e6572

	// SigningKeysLock is named rather than numbered, because this one is worth
	// being able to read in pg_locks.
	SigningKeysLock = Slug + "_signing_keys"
)
