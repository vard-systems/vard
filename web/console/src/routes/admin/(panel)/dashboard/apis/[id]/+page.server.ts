import type { API } from '$lib/api';
import { apiGet, requireAnywhere } from '$lib/server/api';
import type { PageServerLoad } from './$types';

const tabs = ['overview', 'settings', 'scopes', 'applications', 'logs'] as const;

export type ApiTab = (typeof tabs)[number];

export const load: PageServerLoad = async ({ fetch, params, parent, url }) => {
	requireAnywhere((await parent()).admin, 'apis.read');

	const { api } = await apiGet<{ api: API }>(`/admin/apis/${encodeURIComponent(params.id)}`, fetch);

	const asked = url.searchParams.get('tab') as ApiTab | null;
	const tab: ApiTab = asked && tabs.includes(asked) ? asked : 'overview';

	return { api, tab };
};
