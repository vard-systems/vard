import type { ApplicationPage } from '#lib/api/index.js';
import { LIST_PAGE_SIZE } from '#lib/query/index.js';
import { apiGet, requireAnywhere } from '#lib/server/api.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requireAnywhere((await parent()).admin, 'applications.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const type = url.searchParams.get('type') ?? '';

	const query = new URLSearchParams({ limit: String(LIST_PAGE_SIZE) });
	if (search) query.set('search', search);
	if (type) query.set('type', type);

	const page = await apiGet<ApplicationPage>(`/admin/applications?${query}`, fetch);

	return { page, search, type };
};
