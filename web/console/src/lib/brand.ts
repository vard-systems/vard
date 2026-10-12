/**
 * What the project calls itself, and the cookie names built from it. The server keeps the same
 * list in internal/brand and its tests check the two agree; nothing else may spell the name.
 */
export const BRAND = {
	name: 'Loginer',

	slug: 'loginer',

	docsUrl: 'https://loginer.org/docs',
	githubUrl: 'https://github.com/loginer/loginer'
} as const;

export const COOKIES = {
	/** The API's session. Set by the API itself, HttpOnly: the panel never
	    reads it, it only passes it on from a server load. */
	session: `${BRAND.slug}_session`,

	theme: `${BRAND.slug}-theme`,

	sidebar: `${BRAND.slug}-sidebar`,

	branches: `${BRAND.slug}-sidebar-closed`
} as const;
