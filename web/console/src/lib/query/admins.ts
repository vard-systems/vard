import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import { adminsApi, type AdminPage, type AdminPermission, type AdminRole } from '#lib/api/index.js';
import { keys } from './keys';
import { LIST_PAGE_SIZE, nextOffset } from './paging';

export type AdminListParams = { search: string; status: string; role: string };

export function adminsOptions(params: AdminListParams, first: AdminPage) {
	return infiniteQueryOptions({
		queryKey: keys.admins.list(params),
		queryFn: ({ pageParam }) =>
			adminsApi.list({ ...params, offset: pageParam, limit: LIST_PAGE_SIZE }),
		initialPageParam: 0,
		getNextPageParam: (page: AdminPage) => nextOffset(page, page.admins.length),
		initialData: { pages: [first], pageParams: [0] }
	});
}

export function adminRolesOptions(search: string, initial: AdminRole[]) {
	return queryOptions({
		queryKey: keys.admins.roles(search),
		queryFn: async () => (await adminsApi.roles(search)).roles,
		initialData: initial
	});
}

export function adminPermissionsOptions(initial: AdminPermission[]) {
	return queryOptions({
		queryKey: keys.admins.permissions,
		queryFn: async () => (await adminsApi.permissions()).permissions,
		initialData: initial,
		staleTime: Infinity
	});
}
