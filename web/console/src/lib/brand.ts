/**
 * The console's brand: what it calls the product, its mark, the links in its Help menu, its
 * favicon, and every name it builds from the slug. The sign-in pages keep their own
 * (web/id/src/lib/brand.ts) and the server its own (internal/brand); internal/brand's tests hold
 * the three to the same slug and to the cookie names the server reads back. `make rename` renames
 * all three.
 *
 * An installation's own name and logo are not here: an administrator sets them in Settings.
 */
import type { IconName } from '#lib/components/ui/index.js';
import favicon from '#lib/assets/favicon.svg';

type Link = { id: string; label: string; description: string; icon: IconName; href: string };

export const BRAND = {
	/** What a person reads. Never translated. */
	name: 'Vard',

	/** What the names below are built from: lower-case letters and digits, as the server's. */
	slug: 'vard',

	/** The mark drawn where there is no logo: a Remix Icon class. */
	icon: 'ri-shield-keyhole-line',

	/** Imported, so Vite builds it into the app and the URL works in production. */
	favicon,

	/** The Help menu and the command palette offer these, in this order. */
	links: [
		{
			id: 'github',
			label: 'GitHub',
			description: 'Source, issues and releases',
			icon: 'ri-github-fill',
			href: 'https://github.com/vard-systems/vard'
		}
	] satisfies Link[]
} as const;

/** A page's <title>: its parts, most specific first, the empty ones left out, then the product. */
export function pageTitle(...parts: (string | null | undefined)[]): string {
	return [...parts, BRAND.name].filter(Boolean).join(' · ');
}

export const COOKIES = {
	/** The API's session. Set by the API itself, HttpOnly: the panel never
	    reads it, it only passes it on from a server load. */
	session: `${BRAND.slug}_session`,

	theme: `${BRAND.slug}-theme`,

	sidebar: `${BRAND.slug}-sidebar`,

	branches: `${BRAND.slug}-sidebar-closed`
} as const;

/** The $schema an exported login flow is written with; changing it orphans every exported file. */
export const FLOW_SCHEMA = `${BRAND.slug}.login-flow/1` as const;
