import { account } from '#lib/api/index.js';
import { asUser } from '#lib/session.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, url }) => {
	const { sessions } = await asUser(() => account.sessions(fetch), url);

	return { sessions };
};
