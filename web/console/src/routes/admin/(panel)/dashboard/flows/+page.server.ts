import type { LoginFlow, LoginStepSpec } from '$lib/api';
import { apiGet, requirePermission } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'login_flows.read');

	const { flows, step_kinds } = await apiGet<{
		flows: LoginFlow[];
		step_kinds: LoginStepSpec[];
	}>('/admin/login-flows', fetch);

	return {
		flows,
		stepKinds: step_kinds,
		search: url.searchParams.get('search')?.trim() ?? ''
	};
};
