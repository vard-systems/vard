import { error } from '@sveltejs/kit';
import { languagesApi, organizationApi } from '#lib/api/index.js';
import { can } from '#lib/permissions.js';
import { fromApi } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	const { admin } = await parent();

	const mayReadOrganization = can(admin, 'organization.read');
	const mayReadLanguages = can(admin, 'languages.read');

	if (!mayReadOrganization && !mayReadLanguages) {
		error(403, 'Your roles do not allow you to see this page.');
	}

	const [organization, languages] = await Promise.all([
		mayReadOrganization ? fromApi(organizationApi.get(fetch)) : null,
		mayReadLanguages ? fromApi(languagesApi.list(fetch)) : null
	]);

	return { organization: organization?.organization ?? null, languages };
};
