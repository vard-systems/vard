import type { API } from '$lib/api';
import { apiGet, requireAnywhere } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requireAnywhere((await parent()).admin, 'apis.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const query = search ? `?search=${encodeURIComponent(search)}` : '';

	const { apis } = await apiGet<{ apis: API[] }>(`/admin/apis${query}`, fetch);

	return { apis, search };
};
