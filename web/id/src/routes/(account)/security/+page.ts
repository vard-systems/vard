import { account } from '#lib/api/index.js';
import { asUser } from '#lib/server/session.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const { sessions } = await asUser(() => account.sessions(fetch), url);

	return { sessions };
};
