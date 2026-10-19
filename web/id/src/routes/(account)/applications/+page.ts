import { account } from '#lib/api/index.js';
import { asUser } from '#lib/server/session.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, url }) => {
	const { applications } = await asUser(() => account.applications(fetch), url);

	return { applications };
};
