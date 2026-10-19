import { account } from '#lib/api/index.js';
import { asUser } from '#lib/session.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, url }) => {
	const { applications } = await asUser(() => account.applications(fetch), url);

	return { applications };
};
