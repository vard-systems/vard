import { signIn } from '#lib/api/index.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	const clientId = url.searchParams.get('client_id');

	const application = clientId
		? await signIn.application(clientId, fetch).then(
				(answer) => answer.application,
				() => null
			)
		: null;

	return { application };
};
