import type { Role, SSOConnection } from '$lib/api';
import { canAnywhere } from '$lib/permissions';
import { apiGet, requirePermission } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent }) => {
	const { admin } = await parent();
	requirePermission(admin, 'sso.read');

	const mayReadRoles = canAnywhere(admin, 'users.read') || canAnywhere(admin, 'applications.read');

	const [{ connections }, roles] = await Promise.all([
		apiGet<{ connections: SSOConnection[] }>('/admin/sso-connections', fetch),
		mayReadRoles
			? apiGet<{ roles: Role[] }>('/admin/user-roles?limit=200', fetch).then((page) => page.roles)
			: Promise.resolve([] as Role[])
	]);

	return { connections, roles, mayReadRoles };
};
