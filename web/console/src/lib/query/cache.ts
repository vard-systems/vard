import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import {
	cacheApi,
	type CacheDatabaseName,
	type CacheKeyPage,
	type CacheKind,
	type CacheOverview
} from '$lib/api';
import { keys } from './keys';

/** What a listing of keys is narrowed by: the same values the URL carries. */
export type CacheKeyParams = {
	database: CacheDatabaseName;
	kind: CacheKind | '';
	group: string;
	search: string;
};

/** How many keys a page asks for. */
export const CACHE_PAGE_SIZE = 100;

/** Both databases summed up, seeded with what the server rendered. Redis
    changes by the second, so it is read again whenever the page is looked
    at, rather than trusted for a while. */
export function cacheOverviewOptions(initial: CacheOverview) {
	return queryOptions({
		queryKey: keys.cache.overview,
		queryFn: () => cacheApi.overview(),
		initialData: initial,
		staleTime: 0
	});
}

/** One database's keys, a page at a time: Redis walks a database with a
    cursor, so "load more" continues from where the last page stopped. The
    first page is seeded with what the server rendered. */
export function cacheKeysOptions(params: CacheKeyParams, initial?: CacheKeyPage) {
	return infiniteQueryOptions({
		queryKey: keys.cache.keys(params),
		queryFn: ({ pageParam }) =>
			cacheApi.keys(params.database, {
				kind: params.kind,
				group: params.group,
				search: params.search,
				cursor: pageParam,
				limit: CACHE_PAGE_SIZE
			}),
		initialPageParam: '',
		getNextPageParam: (last: CacheKeyPage) => last.cursor || undefined,
		initialData: initial ? { pages: [initial], pageParams: [''] } : undefined,
		staleTime: 0
	});
}

/** One key and what it holds. Read fresh each time: it is about to be edited. */
export function cacheValueOptions(database: CacheDatabaseName, name: string) {
	return queryOptions({
		queryKey: keys.cache.key(database, name),
		queryFn: () => cacheApi.key(database, name),
		staleTime: 0
	});
}
