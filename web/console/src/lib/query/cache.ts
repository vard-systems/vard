import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import {
	cacheApi,
	type CacheDatabaseName,
	type CacheKeyPage,
	type CacheKind,
	type CacheOverview
} from '$lib/api';
import { keys } from './keys';

export type CacheKeyParams = {
	database: CacheDatabaseName;
	kind: CacheKind | '';
	group: string;
	search: string;
};

export const CACHE_PAGE_SIZE = 100;

export function cacheOverviewOptions(initial: CacheOverview) {
	return queryOptions({
		queryKey: keys.cache.overview,
		queryFn: () => cacheApi.overview(),
		initialData: initial,
		staleTime: 0
	});
}

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

export function cacheValueOptions(database: CacheDatabaseName, name: string) {
	return queryOptions({
		queryKey: keys.cache.key(database, name),
		queryFn: () => cacheApi.key(database, name),
		staleTime: 0
	});
}
