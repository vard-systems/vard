import { socialApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'social.read');

	const { providers, kinds } = await fromApi(socialApi.list(fetch));

	return {
		providers,
		kinds,
		search: url.searchParams.get('search')?.trim() ?? '',
		status: url.searchParams.get('status') ?? ''
	};
};
