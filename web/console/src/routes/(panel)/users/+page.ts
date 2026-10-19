import { applicationsApi, rolesApi, usersApi, type Application } from '#lib/api/index.js';
import { APPLICATION_CHOICES_LIMIT, LIST_PAGE_SIZE, ROLE_CHOICES_LIMIT } from '#lib/query/index.js';
import { canAnywhere } from '#lib/permissions.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	const { admin } = await parent();
	requirePermission(admin, 'users.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const verified = url.searchParams.get('verified') ?? '';
	const role = url.searchParams.get('role') ?? '';

	const [page, fields, applications, roles] = await Promise.all([
		fromApi(usersApi.list({ search, verified, role, limit: LIST_PAGE_SIZE }, fetch)),
		fromApi(usersApi.fields(fetch)),
		canAnywhere(admin, 'applications.read')
			? fromApi(applicationsApi.list({ limit: APPLICATION_CHOICES_LIMIT }, fetch)).then(
					(it) => it.applications
				)
			: Promise.resolve<Application[]>([]),
		fromApi(rolesApi.list({ limit: ROLE_CHOICES_LIMIT }, fetch)).then((it) => it.roles)
	]);

	return { page, fields: fields.fields, applications, roles, search, verified, role };
};
