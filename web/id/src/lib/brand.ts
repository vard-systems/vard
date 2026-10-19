/**
 * What the project calls itself, and the cookie names built from it. The server keeps the same
 * list in internal/brand and its tests check the two agree; nothing else may spell the name.
 * The product name is never translated.
 */
export const BRAND = {
	name: 'Loginer',

	slug: 'loginer'
} as const;

export const COOKIES = {
	/** The user's session at the server. Set by the API, HttpOnly: this app
	    never reads it; the browser sends it with every API call. */
	session: `${BRAND.slug}_user_session`,

	theme: `${BRAND.slug}-account-theme`,

	language: `${BRAND.slug}-account-language`
} as const;

export const LANGUAGE_DEPENDENCY = `${BRAND.slug}:language` as const;
