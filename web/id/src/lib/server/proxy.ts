import type { HandleFetch } from '@sveltejs/kit/hooks';
import { API_URL } from '$app/env/private';

const API_PATHS = ['/api/v1/account', '/oauth2', '/.well-known'];

/**
 * Server-side fetches to API paths go straight to API_URL with the reader's cookies and client
 * address, instead of out through the public address.
 */
export const handleFetch: HandleFetch = async ({ event, request, fetch }) => {
	const url = new URL(request.url);

	if (url.origin !== event.url.origin || !API_PATHS.some((path) => url.pathname.startsWith(path))) {
		return fetch(request);
	}

	const headers = new Headers(request.headers);
	const cookie = event.request.headers.get('cookie');
	if (cookie) headers.set('cookie', cookie);
	// Who is really asking, for the API's log. It only believes this from an
	// address in its LOGINER_TRUSTED_PROXIES.
	headers.set('x-forwarded-for', event.getClientAddress());

	const target = new URL(url.pathname + url.search, API_URL);
	const method = request.method;
	const body = method === 'GET' || method === 'HEAD' ? undefined : await request.arrayBuffer();

	return fetch(new Request(target, { method, headers, body, redirect: 'manual' }));
};
