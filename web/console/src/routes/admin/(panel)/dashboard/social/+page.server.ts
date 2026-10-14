import type { SocialProvider, SocialSpec } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/server/api.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'social.read');

	const { providers, kinds } = await apiGet<{
		providers: SocialProvider[];
		kinds: SocialSpec[];
	}>('/admin/social-providers', fetch);

	return {
		providers,
		kinds,
		search: url.searchParams.get('search')?.trim() ?? '',
		status: url.searchParams.get('status') ?? ''
	};
};
