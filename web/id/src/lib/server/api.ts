import type { RequestEvent } from '@sveltejs/kit';
import { API_URL } from '$app/env/private';

/**
 * The paths the API answers. The browser calls them on this app's own origin, so cookies stay
 * host-only and nothing is cross-origin; this server passes them on. internal/api's routing test
 * holds every route to this list.
 */
const API_PATHS = ['/api/v1/account', '/oauth2', '/.well-known'];

// Headers about one connection rather than the request, which a proxy does not pass on.
const HOP_BY_HOP = [
	'connection',
	'keep-alive',
	'host',
	'te',
	'trailer',
	'transfer-encoding',
	'upgrade'
];

/** Whole segments only: `/oauth2` takes `/oauth2/token`, not a page at `/oauth2-help`. */
export function isApiPath(pathname: string): boolean {
	return API_PATHS.some((path) => pathname === path || pathname.startsWith(`${path}/`));
}

/**
 * Passes a request to the API and its answer back unchanged: the browser's own requests, and the
 * ones a load makes while the page is rendered here, which SvelteKit sends through the same hook.
 */
export async function toApi(event: RequestEvent): Promise<Response> {
	const { request, url } = event;

	const headers = new Headers(request.headers);
	for (const name of HOP_BY_HOP) headers.delete(name);

	// The caller's address, resolved here and sent alone: the API believes
	// X-Forwarded-For only from this server (VARD_TRUSTED_PROXIES), so a
	// client's own copy, and the other headers that could claim one, go.
	headers.set('x-forwarded-for', event.getClientAddress());
	headers.delete('x-real-ip');
	headers.delete('forwarded');

	const body =
		request.method === 'GET' || request.method === 'HEAD' ? undefined : await request.arrayBuffer();

	let answer: Response;
	try {
		answer = await fetch(new URL(url.pathname + url.search, API_URL), {
			method: request.method,
			headers,
			body,
			// A redirect is the browser's to follow: an authorization answer
			// carries a code in its Location.
			redirect: 'manual'
		});
	} catch {
		return new Response(null, { status: 502 });
	}

	// fetch has already decoded the body, so the encoding and length it came
	// with no longer describe it.
	const out = new Headers(answer.headers);
	out.delete('content-encoding');
	out.delete('content-length');

	return new Response(answer.body, {
		status: answer.status,
		statusText: answer.statusText,
		headers: out
	});
}
