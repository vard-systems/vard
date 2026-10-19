import { mailApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const [mail, content] = await Promise.all([
		fromApi(mailApi.get(fetch)),
		fromApi(mailApi.content(fetch))
	]);

	return { mail, content };
};
