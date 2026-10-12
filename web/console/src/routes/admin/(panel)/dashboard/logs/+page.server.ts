import { logQuery, type LogPage } from '$lib/api';
import { apiGet, requirePermission } from '$lib/server/api';
import { filterFrom } from '$lib/components/activity/filters';
import { LOGS_PAGE_SIZE } from '$lib/query';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'activity.read');

	const view = filterFrom(url.searchParams);
	const first = await apiGet<LogPage>(
		`/admin/logs${logQuery(view.filter, { limit: String(LOGS_PAGE_SIZE) })}`,
		fetch
	);

	return { view, first };
};
