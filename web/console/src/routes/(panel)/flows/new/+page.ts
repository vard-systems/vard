import { flowsApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'login_flows.write');

	const { flows, step_kinds } = await fromApi(flowsApi.list(fetch));

	return {
		template: url.searchParams.get('template') ?? 'password',
		stepKinds: step_kinds,
		taken: flows.map((one) => one.slug)
	};
};
