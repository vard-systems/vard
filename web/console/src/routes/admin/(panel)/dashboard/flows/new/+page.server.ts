import type { LoginFlow, LoginStepSpec } from '$lib/api';
import { apiGet, requirePermission } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'login_flows.write');

	const { flows, step_kinds } = await apiGet<{ flows: LoginFlow[]; step_kinds: LoginStepSpec[] }>(
		'/admin/login-flows',
		fetch
	);

	return {
		template: url.searchParams.get('template') ?? 'password',
		stepKinds: step_kinds,
		taken: flows.map((one) => one.slug)
	};
};
