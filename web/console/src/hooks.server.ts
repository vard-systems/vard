import type { Handle } from '@sveltejs/kit/hooks';

import { isApiPath, toApi } from '#lib/server/api.js';
import { setPageHeaders } from '#lib/server/headers.js';

export const handle: Handle = async ({ event, resolve }) => {
	if (isApiPath(event.url.pathname)) return toApi(event);

	const response = await resolve(event);
	setPageHeaders(response.headers, event.url);

	return response;
};
