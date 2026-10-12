import type { AdminSession, Overview, OverviewDays } from '$lib/api';
import { can } from '$lib/permissions';
import { apiGet } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	const { admin } = await parent();

	const asked = Number(url.searchParams.get('days'));
	const days: OverviewDays = asked === 7 || asked === 30 || asked === 90 ? asked : 14;

	const [overview, sessions] = await Promise.all([
		can(admin, 'activity.read')
			? apiGet<Overview>(`/admin/overview?days=${days}`, fetch).then(withLists)
			: Promise.resolve(null),
		apiGet<{ sessions: AdminSession[] | null }>('/admin/sessions', fetch)
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
