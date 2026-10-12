import type { UserSessionPage } from '$lib/api';
import { SESSIONS_PAGE_SIZE } from '$lib/query';
import { apiGet, requirePermission } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'users.read');

	const search = url.searchParams.get('search')?.trim() ?? '';
	const user = url.searchParams.get('user') ?? '';
	const email = url.searchParams.get('email') ?? '';

	const query = new URLSearchParams({ limit: String(SESSIONS_PAGE_SIZE) });
	if (search) query.set('search', search);
	if (user) query.set('user', user);

	const page = await apiGet<UserSessionPage>(`/admin/user-sessions?${query}`, fetch);

	return { page, search, user, email };
};
