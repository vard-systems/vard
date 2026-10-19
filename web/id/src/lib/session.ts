import { error, redirect, type Cookies } from '@sveltejs/kit';
import { ApiError, account, type User } from '#lib/api/index.js';
import { COOKIES } from '#lib/brand.js';

export async function currentUser(
	cookies: Cookies,
	fetch: typeof globalThis.fetch
): Promise<User | null> {
	// No cookie, no session: no need to ask. The cookie is the API's, set on
	// this app's origin through the proxy, so the renderer can see it.
	if (!cookies.get(COOKIES.session)) return null;

	try {
		return (await account.me(fetch)).user;
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) return null;
		if (err instanceof ApiError && err.status === 0) error(503, err.message);
		throw err;
	}
}

export async function requireUser(
	cookies: Cookies,
	fetch: typeof globalThis.fetch,
	url: URL
): Promise<User> {
	const user = await currentUser(cookies, fetch);

	if (!user) {
		redirect(303, `/login?next=${encodeURIComponent(url.pathname + url.search)}`);
	}

	return user;
}

export async function asUser<T>(call: () => Promise<T>, url: URL): Promise<T> {
	try {
		return await call();
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) {
			redirect(303, `/login?next=${encodeURIComponent(url.pathname)}`);
		}
		throw err;
	}
}
