import type { AdminPermission, AdminRole } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/server/api.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const query = search ? `?search=${encodeURIComponent(search)}` : '';

	const [roles, catalog] = await Promise.all([
		apiGet<{ roles: AdminRole[] }>(`/admin/admin-roles${query}`, fetch),
		apiGet<{ permissions: AdminPermission[] }>('/admin/admin-permissions', fetch)
	]);

	return { roles: roles.roles, catalog: catalog.permissions, search };
};
