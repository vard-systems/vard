import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import { applicationsApi, type Application, type ApplicationPage } from '$lib/api';
import { keys } from './keys';
import { LIST_PAGE_SIZE, nextOffset } from './paging';

/** How many applications the pickers ask for: the most the API gives. */
export const APPLICATION_CHOICES_LIMIT = 500;

/** What the list is asked for, all of it from the URL. */
export type ApplicationListParams = { search: string; type: string };

/** The applications matching a search, as pages, seeded with the first one
    the server rendered. */
export function applicationsOptions(params: ApplicationListParams, first: ApplicationPage) {
	return infiniteQueryOptions({
		queryKey: keys.applications.list(params),
		queryFn: ({ pageParam }) =>
			applicationsApi.list({ ...params, offset: pageParam, limit: LIST_PAGE_SIZE }),
		initialPageParam: 0,
		getNextPageParam: (page: ApplicationPage) => nextOffset(page, page.applications.length),
		initialData: { pages: [first], pageParams: [0] }
	});
}

/** Every application the administrator can see, for the pickers that offer
    them: the roles page, and the user and administrator panels. */
export function applicationChoicesOptions(initial: Application[]) {
	return queryOptions({
		queryKey: keys.applications.choices,
		queryFn: async () =>
			(await applicationsApi.list({ limit: APPLICATION_CHOICES_LIMIT })).applications,
		initialData: initial
	});
}

/** What one application may do with each API. */
export function applicationAccessOptions(id: string) {
	return queryOptions({
		queryKey: keys.applications.access(id),
		queryFn: async () => (await applicationsApi.apiAccess(id)).apis
	});
}
