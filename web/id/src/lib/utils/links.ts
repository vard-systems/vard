import { resolve } from '$app/paths';
import type { Path } from '$app/types';

export type AuthPage = '/login' | '/register' | '/forgot-password' | '/sso';

export function authHref(page: AuthPage, request: string | null): string {
	// resolve() reads a leading slash as a route id; a page is a plain path.
	const path = resolve(page.slice(1) as Path);
	return request ? `${path}?request=${encodeURIComponent(request)}` : path;
}

/** The second-factor page, for a sign-in that waits for one: the sign-in it finishes is named by a
    cookie, so the address carries only where to go after. */
export function secondFactorHref(request: string | null, next: string | null): string {
	const query = new URLSearchParams();
	if (request) query.set('request', request);
	if (next && next !== '/') query.set('next', next);

	const search = query.toString();
	return `${resolve('second-factor')}${search ? `?${search}` : ''}`;
}

type StartOptions = { request?: string | null; next?: string | null; email?: string };

/** The server's own path that sends the browser on to an SSO connection. */
export function ssoHref(slug: string, options: StartOptions): string {
	return startHref('sso', slug, options);
}

/** The server's own path that sends the browser on to a social provider. */
export function socialHref(slug: string, options: StartOptions): string {
	return startHref('social', slug, options);
}

function startHref(kind: 'sso' | 'social', slug: string, options: StartOptions): string {
	const query = new URLSearchParams();
	if (options.request) query.set('request', options.request);
	if (options.next) query.set('next', options.next);
	if (options.email) query.set('login_hint', options.email);

	const search = query.toString();
	return `/oauth2/${kind}/${encodeURIComponent(slug)}/start${search ? `?${search}` : ''}`;
}

/** Sends the browser on to where the API said: usually back to an
    application, on another origin, so it is a full navigation. */
export function leaveTo(location: string): void {
	window.location.assign(location);
}
