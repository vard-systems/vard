import type { Handle } from '@sveltejs/kit/hooks';

import { COOKIES } from '#lib/brand.js';

export { handleFetch } from '#lib/server/proxy.js';

/**
 * Writes the reader's chosen theme into <html> before it is sent, so there is no
 * light-then-dark flash; with no choice, prefers-color-scheme in tokens.css decides. <html
 * lang> is always English: the panel is not translated.
 */
export const handle: Handle = async ({ event, resolve }) => {
	const saved = event.cookies.get(COOKIES.theme);
	const theme = saved === 'dark' || saved === 'light' ? saved : null;
	const attribute = theme ? `data-theme="${theme}"` : '';

	const response = await resolve(event, {
		transformPageChunk: ({ html }) =>
			html.replace('__THEME__', attribute).replace('__LANG__', 'en'),
		// The body text's face is fetched with the page, so the text is not
		// drawn in a fallback first and redrawn when it arrives.
		preload: ({ type, path }) =>
			type === 'font' ? /chirp-latin-400-normal\./.test(path) : type === 'js' || type === 'css'
	});

	// The panel takes an administrator's password and acts on their behalf,
	// so no other site may frame it: a transparent frame over a fake page is
	// how clickjacking gets a click or a password it should not.
	response.headers.set('X-Frame-Options', 'DENY');
	response.headers.set('Content-Security-Policy', "frame-ancestors 'none'");
	// The edge sets these too; the panel sets them itself so that it is just
	// as safe behind any other proxy.
	response.headers.set('Referrer-Policy', 'no-referrer');
	response.headers.set('X-Content-Type-Options', 'nosniff');

	// Every page is drawn for one person — their account, or a form with a
	// one-time token in it — so no cache may keep it, and Back after signing
	// out does not show it again.
	if (response.headers.get('content-type')?.startsWith('text/html')) {
		response.headers.set('Cache-Control', 'no-store');
	}

	return response;
};
