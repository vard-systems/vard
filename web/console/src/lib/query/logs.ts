import { infiniteQueryOptions } from '@tanstack/svelte-query';

import { activityApi, type LogFilter, type LogPage } from '#lib/api/index.js';
import { keys } from './keys';

export const LOGS_PAGE_SIZE = 50;

export function logsOptions(filter: LogFilter, first: LogPage) {
	return infiniteQueryOptions({
		queryKey: keys.logs.list(filter),
		queryFn: ({ pageParam }) => activityApi.logs(filter, pageParam, LOGS_PAGE_SIZE),
		initialPageParam: '',
		getNextPageParam: (page: LogPage) => page.next || undefined,
		initialData: { pages: [first], pageParams: [''] }
	});
}
