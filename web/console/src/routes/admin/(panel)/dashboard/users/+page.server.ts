import type { Application, ApplicationPage, RolePage, UserField, UserPage } from '$lib/api';
import { APPLICATION_CHOICES_LIMIT, LIST_PAGE_SIZE, ROLE_CHOICES_LIMIT } from '$lib/query';
import { canAnywhere } from '$lib/permissions';
import { apiGet, requirePermission } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	const { admin } = await parent();
	requirePermission(admin, 'users.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const verified = url.searchParams.get('verified') ?? '';
	const role = url.searchParams.get('role') ?? '';

	const query = new URLSearchParams({ limit: String(LIST_PAGE_SIZE) });
	if (search) query.set('search', search);
	if (verified === 'true' || verified === 'false') query.set('verified', verified);
	if (role) query.set('role', role);

	const seesApplications = canAnywhere(admin, 'applications.read');

	const [page, fields, applications, roles] = await Promise.all([
		apiGet<UserPage>(`/admin/users?${query}`, fetch),
		apiGet<{ fields: UserField[] }>('/admin/user-fields', fetch),
		seesApplications
			? apiGet<ApplicationPage>(
					`/admin/applications?limit=${APPLICATION_CHOICES_LIMIT}`,
					fetch
				).then((it) => it.applications)
			: Promise.resolve<Application[]>([]),
		apiGet<RolePage>(`/admin/user-roles?limit=${ROLE_CHOICES_LIMIT}`, fetch).then((it) => it.roles)
	]);

	return { page, fields: fields.fields, applications, roles, search, verified, role };
};
