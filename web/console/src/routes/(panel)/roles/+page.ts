import { error } from '@sveltejs/kit';
import { apisApi, applicationsApi, rolesApi, type Application } from '#lib/api/index.js';
import { can, canAnywhere } from '#lib/permissions.js';
import { APPLICATION_CHOICES_LIMIT, LIST_PAGE_SIZE, ROLE_CHOICES_LIMIT } from '#lib/query/index.js';
import { fromApi } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	const { admin } = await parent();

	if (!canAnywhere(admin, 'users.read') && !canAnywhere(admin, 'applications.read')) {
		error(403, 'Your roles do not allow you to see this page.');
	}

	const seesApplications = canAnywhere(admin, 'applications.read');

	const tab: 'global' | 'application' =
		url.searchParams.get('tab') === 'application' && seesApplications ? 'application' : 'global';
	const search = url.searchParams.get('search')?.trim() ?? '';
	const isDefault = url.searchParams.get('default') ?? '';

	const [apis, applications, choices] = await Promise.all([
		can(admin, 'apis.read') ? fromApi(apisApi.list('', fetch)).then((it) => it.apis) : null,
		seesApplications
			? fromApi(applicationsApi.list({ limit: APPLICATION_CHOICES_LIMIT }, fetch)).then(
					(it) => it.applications
				)
			: Promise.resolve<Application[]>([]),
		fromApi(rolesApi.list({ limit: ROLE_CHOICES_LIMIT }, fetch))
	]);

	const named = url.searchParams.get('application') ?? '';
	const application =
		tab === 'application' && applications.some((app) => app.id === named) ? named : '';

	const page = await fromApi(
		rolesApi.list(
			{ scope: tab, application, search, default: isDefault, limit: LIST_PAGE_SIZE },
			fetch
		)
	);

	return {
		page,
		applications,
		apis,
		choices: choices.roles,
		tab,
		application,
		search,
		isDefault
	};
};
