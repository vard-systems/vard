import { logQuery, type LogPage } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/api/load.js';
import { filterFrom } from '#lib/components/activity/filters.js';
import { LOGS_PAGE_SIZE } from '#lib/query/index.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'activity.read');

	const view = filterFrom(url.searchParams);
	const first = await apiGet<LogPage>(
		`/admin/logs${logQuery(view.filter, { limit: String(LOGS_PAGE_SIZE) })}`,
		fetch
	);

	return { view, first };
};
