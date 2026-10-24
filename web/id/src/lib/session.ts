import { error, redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { ApiError, account, type User } from '#lib/api/index.js';
import { safeNext } from '#lib/utils/next.js';

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
	if (!user) toSignIn(url);

	return user;
}

/**
 * Where a sign-in page goes on to: its checked `?next=`. A browser that is
 * already signed in, and is not signing in to an application, goes straight
 * there.
 */
export async function nextUnlessSignedIn(
	fetch: typeof globalThis.fetch,
	url: URL
): Promise<string> {
	const next = safeNext(url.searchParams.get('next'));

	if (!url.searchParams.get('request') && (await currentUser(fetch))) {
		redirect(303, next);
	}

	return next;
}

/** `call`, or the sign-in page when the session ended in the meantime. */
export async function asUser<T>(call: () => Promise<T>, url: URL): Promise<T> {
	try {
		return await call();
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) toSignIn(url);
		throw err;
	}
}

/** The sign-in page, coming back to the whole of `url` (its query too) once signed in. */
function toSignIn(url: URL): never {
	redirect(
		303,
		`${resolve('/(auth)/login')}?next=${encodeURIComponent(url.pathname + url.search)}`
	);
}
