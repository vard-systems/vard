import type { Handle } from '@sveltejs/kit/hooks';

import { BASE } from '#lib/i18n/index.js';
import { isApiPath, toApi } from '#lib/server/api.js';
import { setPageHeaders } from '#lib/server/headers.js';

export const handle: Handle = async ({ event, resolve }) => {
	if (isApiPath(event.url.pathname)) return toApi(event);

	// The root layout decides the language; it is written into <html> here
	// so a screen reader has it from the first byte.
	const response = await resolve(event, {
		transformPageChunk: ({ html }) => html.replace('%lang%', event.locals.language ?? BASE)
	});
	setPageHeaders(response.headers);

	return response;
};
