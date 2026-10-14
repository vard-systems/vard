import { infiniteQueryOptions } from '@tanstack/svelte-query';

import { sessionsApi, type UserSessionPage } from '#lib/api/index.js';
import { keys } from './keys';

export const SESSIONS_PAGE_SIZE = 50;

export type SessionListParams = { search: string; user: string };

export function sessionsOptions(params: SessionListParams, first: UserSessionPage) {
	return infiniteQueryOptions({
		queryKey: keys.sessions.list(params),
		queryFn: ({ pageParam }) =>
			sessionsApi.list({ ...params, after: pageParam, limit: SESSIONS_PAGE_SIZE }),
		initialPageParam: '',
		getNextPageParam: (page: UserSessionPage) => page.next || undefined,
		initialData: { pages: [first], pageParams: [''] }
	});
}
