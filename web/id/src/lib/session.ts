import { error, redirect } from '@sveltejs/kit';
import { ApiError, account, type User } from '#lib/api/index.js';

/** Who is signed in, asked of the API: the session cookie is HttpOnly, so
 *  only the answer to /account/me can say whether there is one. */
export async function currentUser(fetch: typeof globalThis.fetch): Promise<User | null> {
	try {
		return (await account.me(fetch)).user;
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) return null;
		if (err instanceof ApiError && err.status === 0) error(503, err.message);
		throw err;
	}
}

export async function requireUser(fetch: typeof globalThis.fetch, url: URL): Promise<User> {
	const user = await currentUser(fetch);

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
