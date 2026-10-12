import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import { usersApi, type UserField, type UserPage } from '$lib/api';
import { keys } from './keys';
import { LIST_PAGE_SIZE, nextOffset } from './paging';

/** What the list is asked for: the search box, the verified filter and the
    role filter, all of which live in the URL. */
export type UserListParams = { search: string; verified: string; role: string };

/** Users matching a search as pages; `first` is the server-rendered page for this key only. */
export function usersOptions(params: UserListParams, first: UserPage) {
	return infiniteQueryOptions({
		queryKey: keys.users.list(params),
		queryFn: ({ pageParam }) =>
			usersApi.list({ ...params, offset: pageParam, limit: LIST_PAGE_SIZE }),
		initialPageParam: 0,
		getNextPageParam: (page: UserPage) => nextOffset(page, page.users.length),
		initialData: { pages: [first], pageParams: [0] }
	});
}

/** The fields every user record has. They change rarely, and everything on
    the page needs them, so they are one query the whole page shares. */
export function userFieldsOptions(initial: UserField[]) {
	return queryOptions({
		queryKey: keys.users.fields,
		queryFn: async () => (await usersApi.fields()).fields,
		initialData: initial
	});
}

/** Every role one user holds, with where each comes from. */
export function roleMappingsOptions(userId: string) {
	return queryOptions({
		queryKey: keys.users.mappings(userId),
		queryFn: async () => (await usersApi.roleMappings(userId)).roles
	});
}

/** How many users a picker offers for a typed name. */
export const USER_SEARCH_LIMIT = 6;

/** The first few users matching a typed name, for pickers. */
export function userSearchOptions(term: string) {
	return queryOptions({
		queryKey: keys.users.search(term),
		queryFn: async () => (await usersApi.list({ search: term })).users.slice(0, USER_SEARCH_LIMIT)
	});
}
