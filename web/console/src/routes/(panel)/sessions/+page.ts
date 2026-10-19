import { sessionsApi } from '#lib/api/index.js';
import { SESSIONS_PAGE_SIZE } from '#lib/query/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'users.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const user = url.searchParams.get('user') ?? '';
	const email = url.searchParams.get('email') ?? '';

	const page = await fromApi(sessionsApi.list({ search, user, limit: SESSIONS_PAGE_SIZE }, fetch));

	return { page, search, user, email };
};
