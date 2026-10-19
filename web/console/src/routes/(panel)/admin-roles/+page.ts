import { adminsApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const search = url.searchParams.get('search')?.trim() ?? '';

	const [roles, catalog] = await Promise.all([
		fromApi(adminsApi.roles(search, fetch)),
		fromApi(adminsApi.permissions(fetch))
	]);

	return { roles: roles.roles, catalog: catalog.permissions, search };
};
