import { adminsApi, applicationsApi } from '#lib/api/index.js';
import { APPLICATION_CHOICES_LIMIT, LIST_PAGE_SIZE } from '#lib/query/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const status = url.searchParams.get('status') ?? '';
	const role = url.searchParams.get('role') ?? '';

	const [page, roles, catalog, applications, security] = await Promise.all([
		fromApi(adminsApi.list({ search, status, role, limit: LIST_PAGE_SIZE }, fetch)),
		fromApi(adminsApi.roles('', fetch)),
		fromApi(adminsApi.permissions(fetch)),
		fromApi(applicationsApi.list({ limit: APPLICATION_CHOICES_LIMIT }, fetch)),
		fromApi(adminsApi.security(fetch))
	]);

	return {
		page,
		roles: roles.roles,
		catalog: catalog.permissions,
		applications: applications.applications,
		security,
		search,
		status,
		role
	};
};
