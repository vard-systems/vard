import { flowsApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'login_flows.read');

	const { flows, step_kinds } = await fromApi(flowsApi.list(fetch));

	return {
		flows,
		stepKinds: step_kinds,
		search: url.searchParams.get('search')?.trim() ?? ''
	};
};
