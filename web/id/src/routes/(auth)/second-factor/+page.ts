import { signIn } from '#lib/api/index.js';
import { safeNext } from '#lib/utils/next.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	// Fails when the cookie naming the sign-in is gone or the sign-in has
	// timed out; the page then says so.
	const challenge = await signIn.secondFactor(fetch).then(
		(answer) => answer.second_factor,
		() => null
	);

	return { challenge, next: safeNext(url.searchParams.get('next')) };
};
