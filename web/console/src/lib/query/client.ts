import { browser } from '$app/env';
import { QueryClient } from '@tanstack/svelte-query';

/**
 * A query client per visitor, created in the root layout. Each page's load has just fetched its
 * data, so it is considered fresh for 30 seconds and refetched on tab focus after that.
 */
export function createQueryClient(): QueryClient {
	return new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 30_000,
				refetchOnWindowFocus: browser,
				retry: browser ? 1 : false
			}
		}
	});
}
