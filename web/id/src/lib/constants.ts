/** How long a remembered choice (language, theme) lasts, in seconds. */
export const ONE_YEAR = 60 * 60 * 24 * 365;

/**
 * The lengths the API accepts (internal/api/account/request.go and
 * oidc.MinPasswordLength). The server checks them regardless; the pages use
 * them so a form says so before it is sent.
 */
export const LIMITS = {
	name: 100,
	email: 255,
	code: 16,
	passwordMin: 8
} as const;
