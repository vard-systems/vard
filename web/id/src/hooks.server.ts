import type { Handle } from '@sveltejs/kit/hooks';

import { COOKIES } from '#lib/brand.js';
import { BASE } from '#lib/i18n/index.js';
import { isApiPath, toApi } from '#lib/server/api.js';
import { setPageHeaders } from '#lib/server/headers.js';
import { isTheme } from '#lib/theme.js';
import { readChoice } from '#lib/utils/cookies.js';

export const handle: Handle = async ({ event, resolve }) => {
	if (isApiPath(event.url.pathname)) return toApi(event);

	// The root layout decides the language, and the reader's cookie the
	// theme; both are written into <html> here, so a screen reader has the
	// language from the first byte and a dark page never flashes light.
	const theme = readChoice(event.cookies, COOKIES.theme, event.url);
	const response = await resolve(event, {
		transformPageChunk: ({ html }) =>
			html
				.replace('%lang%', event.locals.language ?? BASE)
				.replace('%theme%', isTheme(theme) ? theme : '')
	});
	setPageHeaders(response.headers, event.url);

	return response;
};
