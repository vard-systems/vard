import { account } from '#lib/api/index.js';
import { asUser } from '#lib/session.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, url }) => {
	const [{ sessions }, { two_step: twoStep }] = await Promise.all([
		asUser(() => account.sessions(fetch), url),
		asUser(() => account.twoStep(fetch), url)
	]);

	return { sessions, twoStep };
};
