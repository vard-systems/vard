import type { API } from '#lib/api/index.js';
import { apiGet, requireAnywhere } from '#lib/api/load.js';
import type { PageLoad } from './$types';

const tabs = ['overview', 'settings', 'scopes', 'applications', 'logs'] as const;

export type ApiTab = (typeof tabs)[number];

export const load: PageLoad = async ({ fetch, params, parent, url }) => {
	requireAnywhere((await parent()).admin, 'apis.read');

	const { api } = await apiGet<{ api: API }>(`/admin/apis/${encodeURIComponent(params.id)}`, fetch);

	const asked = url.searchParams.get('tab') as ApiTab | null;
	const tab: ApiTab = asked && tabs.includes(asked) ? asked : 'overview';

	return { api, tab };
};
