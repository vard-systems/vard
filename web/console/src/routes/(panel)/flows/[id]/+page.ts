import { error } from '@sveltejs/kit';
import { flowsApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, params }) => {
	requirePermission((await parent()).admin, 'login_flows.read');

	const { flows, step_kinds } = await fromApi(flowsApi.list(fetch));

	const flow = flows.find((one) => one.id === params.id);
	if (!flow) error(404, 'There is no such login flow.');

	return { flow, stepKinds: step_kinds, taken: flows.map((one) => one.slug) };
};
