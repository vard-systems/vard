import { account, signIn } from '#lib/api/index.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	const clientId = url.searchParams.get('client_id');

	const [application, user] = await Promise.all([
		clientId
			? signIn.application(clientId, fetch).then(
					(answer) => answer.application,
					() => null
				)
			: null,
		account.me(fetch).then(
			(answer) => answer.user,
			() => null
		)
	]);

	return { application, user };
};
