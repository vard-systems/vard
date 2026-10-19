import { rolesApi, ssoApi, type Role } from '#lib/api/index.js';
import { canAnywhere } from '#lib/permissions.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	const { admin } = await parent();
	requirePermission(admin, 'sso.read');

	const mayReadRoles = canAnywhere(admin, 'users.read') || canAnywhere(admin, 'applications.read');

	const [{ connections }, roles] = await Promise.all([
		fromApi(ssoApi.list(fetch)),
		mayReadRoles
			? fromApi(rolesApi.list({ limit: 200 }, fetch)).then((page) => page.roles)
			: Promise.resolve([] as Role[])
	]);

	return { connections, roles, mayReadRoles };
};
