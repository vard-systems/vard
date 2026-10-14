import { resolve } from '$app/paths';
import type { Path } from '$app/types';

export type AuthPage = '/login' | '/register' | '/forgot-password' | '/sso';

export function authHref(page: AuthPage, request: string | null): string {
	// resolve() reads a leading slash as a route id; a page is a plain path.
	const path = resolve(page.slice(1) as Path);
	return request ? `${path}?request=${encodeURIComponent(request)}` : path;
}

export function ssoHref(
	slug: string,
	options: { request?: string | null; next?: string | null; email?: string }
): string {
	const query = new URLSearchParams();
	if (options.request) query.set('request', options.request);
	if (options.next) query.set('next', options.next);
	if (options.email) query.set('login_hint', options.email);

	const search = query.toString();
	return `/oauth2/sso/${encodeURIComponent(slug)}/start${search ? `?${search}` : ''}`;
}

/** Sends the browser on to where the API said: usually back to an
    application, on another origin, so it is a full navigation. */
export function leaveTo(location: string): void {
	window.location.assign(location);
}
