/**
 * The sign-in pages' brand: what they call the product, its mark, their favicon, and every name
 * they build from the slug. The console keeps its own (web/console/src/lib/brand.ts) and the
 * server its own (internal/brand); internal/brand's tests hold the three to the same slug and to
 * the cookie names the server reads back. `make rename` renames all three.
 *
 * An installation's own name, logo, terms and privacy links are not here: an administrator sets
 * them in the console, and these pages show those first. The product name is never translated.
 */
import favicon from '#lib/assets/favicon.svg';

export const BRAND = {
	/** What a person reads: "Secured by …", and the recovery codes' file name. */
	name: 'Loginer',

	/** What the names below are built from: lower-case letters and digits, as the server's. */
	slug: 'loginer',

	/** The mark drawn where there is no logo: a Remix Icon class. */
	icon: 'ri-shield-keyhole-line',

	/** Imported, so Vite builds it into the app and the URL works in production. */
	favicon
} as const;

export const COOKIES = {
	/** The user's session at the server. Set by the API, HttpOnly: this app
	    never reads it; the browser sends it with every API call. */
	session: `${BRAND.slug}_user_session`,

	theme: `${BRAND.slug}-account-theme`,

	language: `${BRAND.slug}-account-language`
} as const;

export const LANGUAGE_DEPENDENCY = `${BRAND.slug}:language` as const;
