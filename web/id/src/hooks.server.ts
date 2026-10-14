import type { Handle } from '@sveltejs/kit/hooks';

import { COOKIES } from '#lib/brand.js';
import { BASE } from '#lib/i18n/index.js';

export { handleFetch } from '#lib/server/proxy.js';

/**
 * Writes the reader's theme and language into <html> before sending, so there is no flash and
 * screen readers get the right lang (the root layout decides the language in `locals`). Also
 * forbids framing on every page, since each takes a password or acts for a signed-in user.
 */
export const handle: Handle = async ({ event, resolve }) => {
	const saved = event.cookies.get(COOKIES.theme);
	const theme = saved === 'dark' || saved === 'light' ? saved : null;

	const attributes = theme ? `data-theme="${theme}"` : '';

	const response = await resolve(event, {
		transformPageChunk: ({ html }) =>
			html.replace('__THEME__', attributes).replace('__LANG__', event.locals.language ?? BASE)
	});

	response.headers.set('X-Frame-Options', 'DENY');
	response.headers.set('Content-Security-Policy', "frame-ancestors 'none'");
	response.headers.set('Referrer-Policy', 'no-referrer');
	response.headers.set('X-Content-Type-Options', 'nosniff');

	return response;
};
