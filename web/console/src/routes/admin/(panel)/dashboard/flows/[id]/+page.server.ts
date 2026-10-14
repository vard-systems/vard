import { error } from '@sveltejs/kit';
import type { LoginFlow, LoginStepSpec } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/server/api.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, params }) => {
	requirePermission((await parent()).admin, 'login_flows.read');

	const { flows, step_kinds } = await apiGet<{ flows: LoginFlow[]; step_kinds: LoginStepSpec[] }>(
		'/admin/login-flows',
		fetch
	);

	const flow = flows.find((one) => one.id === params.id);
	if (!flow) error(404, 'There is no such login flow.');

	return { flow, stepKinds: step_kinds, taken: flows.map((one) => one.slug) };
};
