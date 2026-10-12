import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import { usersApi, type UserField, type UserPage } from '$lib/api';
import { keys } from './keys';
import { LIST_PAGE_SIZE, nextOffset } from './paging';

export type UserListParams = { search: string; verified: string; role: string };

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

export function userFieldsOptions(initial: UserField[]) {
	return queryOptions({
		queryKey: keys.users.fields,
		queryFn: async () => (await usersApi.fields()).fields,
		initialData: initial
	});
}

export function roleMappingsOptions(userId: string) {
	return queryOptions({
		queryKey: keys.users.mappings(userId),
		queryFn: async () => (await usersApi.roleMappings(userId)).roles
	});
}

export const USER_SEARCH_LIMIT = 6;

export function userSearchOptions(term: string) {
	return queryOptions({
		queryKey: keys.users.search(term),
		queryFn: async () => (await usersApi.list({ search: term })).users.slice(0, USER_SEARCH_LIMIT)
	});
}
