import { apisApi } from '#lib/api/index.js';
import { fromApi, requireAnywhere } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requireAnywhere((await parent()).admin, 'apis.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const { apis } = await fromApi(apisApi.list(search, fetch));

	return { apis, search };
};
