import {
	cacheApi,
	type CacheDatabaseName,
	type CacheKeyPage,
	type CacheKind
} from '#lib/api/index.js';
import { CACHE_PAGE_SIZE } from '#lib/query/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

const kinds: CacheKind[] = ['entry', 'stale', 'generation', 'session', 'ratelimit', 'other'];

export const load: PageLoad = async ({ fetch, parent, url }) => {
	requirePermission((await parent()).admin, 'super_admin');

	const database: CacheDatabaseName =
		url.searchParams.get('database') === 'sessions' ? 'sessions' : 'cache';
	const asked = url.searchParams.get('kind') ?? '';
	const kind: CacheKind | '' = kinds.includes(asked as CacheKind) ? (asked as CacheKind) : '';
	const group = url.searchParams.get('group')?.trim() ?? '';
	const search = url.searchParams.get('search')?.trim() ?? '';

	const overview = await fromApi(cacheApi.overview(fetch));

	const first: CacheKeyPage = overview.configured
		? await fromApi(cacheApi.keys(database, { kind, group, search, limit: CACHE_PAGE_SIZE }, fetch))
		: { keys: [], cursor: '' };

	return { overview, first, database, kind, group, search };
};
