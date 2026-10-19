import { activityApi, adminApi, type Overview, type OverviewDays } from '#lib/api/index.js';
import { can } from '#lib/permissions.js';
import { fromApi } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent, url }) => {
	const { admin } = await parent();

	const asked = Number(url.searchParams.get('days'));
	const days: OverviewDays = asked === 7 || asked === 30 || asked === 90 ? asked : 14;

	const [overview, sessions] = await Promise.all([
		can(admin, 'activity.read')
			? fromApi(activityApi.overview(days, fetch)).then(withLists)
			: Promise.resolve(null),
		fromApi(adminApi.profileSessions(fetch))
	]);

	return { days, overview, sessions: sessions.sessions ?? [] };
};

function withLists(overview: Overview): Overview {
	return {
		...overview,
		daily: overview.daily ?? [],
		top_actors: overview.top_actors ?? [],
		activity: overview.activity ?? []
	};
}
