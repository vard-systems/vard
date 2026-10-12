import { error } from '@sveltejs/kit';
import type { API, Application, ApplicationPage, RolePage } from '$lib/api';
import { can, canAnywhere } from '$lib/permissions';
import { APPLICATION_CHOICES_LIMIT, LIST_PAGE_SIZE, ROLE_CHOICES_LIMIT } from '$lib/query';
import { apiGet } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	const { admin } = await parent();

	if (!canAnywhere(admin, 'users.read') && !canAnywhere(admin, 'applications.read')) {
		error(403, 'Your roles do not allow you to see this page.');
	}

	const seesApplications = canAnywhere(admin, 'applications.read');

	const tab =
		url.searchParams.get('tab') === 'application' && seesApplications ? 'application' : 'global';
	const search = url.searchParams.get('search')?.trim() ?? '';
	const isDefault = url.searchParams.get('default') ?? '';

	const apis = can(admin, 'apis.read')
		? (await apiGet<{ apis: API[] }>('/admin/apis', fetch)).apis
		: null;

	const [applications, choices] = await Promise.all([
		seesApplications
			? apiGet<ApplicationPage>(
					`/admin/applications?limit=${APPLICATION_CHOICES_LIMIT}`,
					fetch
				).then((it) => it.applications)
			: Promise.resolve<Application[]>([]),
		apiGet<RolePage>(`/admin/user-roles?limit=${ROLE_CHOICES_LIMIT}`, fetch)
	]);

	const named = url.searchParams.get('application') ?? '';
	const application =
		tab === 'application' && applications.some((app) => app.id === named) ? named : '';

	const query = new URLSearchParams({ scope: tab, limit: String(LIST_PAGE_SIZE) });
	if (application) query.set('application', application);
	if (search) query.set('search', search);
	if (isDefault === 'true' || isDefault === 'false') query.set('default', isDefault);

	const page = await apiGet<RolePage>(`/admin/user-roles?${query}`, fetch);

	return {
		page,
		applications,
		apis,
		choices: choices.roles,
		tab: tab as 'global' | 'application',
		application,
		search,
		isDefault
	};
};
