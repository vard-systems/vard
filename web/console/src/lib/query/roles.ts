import { infiniteQueryOptions, queryOptions } from '@tanstack/svelte-query';

import { rolesApi, type Role, type RolePage } from '$lib/api';
import { keys } from './keys';
import { LIST_PAGE_SIZE, nextOffset } from './paging';

export type RoleListParams = {
	scope: 'global' | 'application';
	application: string;
	search: string;
	isDefault: string;
};

export const ROLE_CHOICES_LIMIT = 500;

export function rolesOptions(params: RoleListParams, first: RolePage) {
	return infiniteQueryOptions({
		queryKey: keys.roles.list(params),
		queryFn: ({ pageParam }) =>
			rolesApi.list({
				scope: params.scope,
				application: params.application || undefined,
				search: params.search,
				default: params.isDefault,
				offset: pageParam,
				limit: LIST_PAGE_SIZE
			}),
		initialPageParam: 0,
		getNextPageParam: (page: RolePage) => nextOffset(page, page.roles.length),
		initialData: { pages: [first], pageParams: [0] }
	});
}

export function roleChoicesOptions(initial: Role[]) {
	return queryOptions({
		queryKey: keys.roles.choices,
		queryFn: async () => (await rolesApi.list({ limit: ROLE_CHOICES_LIMIT })).roles,
		initialData: initial
	});
}
