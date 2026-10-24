import { account, signIn } from '#lib/api/index.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	const request = url.searchParams.get('request');

	// Either fails for a sign-in that has ended or a browser that is not
	// signed in; the page then says the sign-in is over.
	const [consent, user] = await Promise.all([
		request
			? signIn.consent(request, fetch).then(
					(answer) => answer.consent,
					() => null
				)
			: null,
		account.me(fetch).then(
			(answer) => answer.user,
			() => null
		)
	]);

	return { consent, user };
};
