import { getRequestEvent } from '$app/server';
import type { Handle } from '@sveltejs/kit/hooks';

import { useRequestFetch } from '#lib/api/client.js';
import { COOKIES } from '#lib/brand.js';
import { isApiPath, toApi } from '#lib/server/api.js';
import { setPageHeaders } from '#lib/server/headers.js';
import { isTheme } from '#lib/theme.js';
import { readChoice } from '#lib/utils/cookies.js';

// A form action calls the admin API with the request's own fetch, so the
// call carries the administrator's cookies and comes back through toApi. The
// admin API takes a change only from this origin, and says so by the Origin
// header, which a call made here does not have: it is set to this origin,
// which is true, because every action (`act`) refuses a form posted from
// anywhere else before it calls.
useRequestFetch(() => {
	let event;
	try {
		event = getRequestEvent();
	} catch {
		return undefined;
	}
	const { fetch, url } = event;
	return (input, init) => {
		const headers = new Headers(init?.headers);
		headers.set('origin', url.origin);
		return fetch(input, { ...init, headers });
	};
});

export const handle: Handle = async ({ event, resolve }) => {
	if (isApiPath(event.url.pathname)) return toApi(event);

	// The reader's theme is written into <html> as the page is rendered, so
	// a dark page never flashes light and no script has to guess it.
	const theme = readChoice(event.cookies, COOKIES.theme, event.url);
	const response = await resolve(event, {
		transformPageChunk: ({ html }) => html.replace('%theme%', isTheme(theme) ? theme : '')
	});
	setPageHeaders(response.headers, event.url);

	return response;
};
