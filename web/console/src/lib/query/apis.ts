import { queryOptions } from '@tanstack/svelte-query';

import { apisApi, type API } from '$lib/api';
import { keys } from './keys';

export function apisOptions(search: string, initial: API[]) {
	return queryOptions({
		queryKey: keys.apis.list(search),
		queryFn: async () => (await apisApi.list(search)).apis,
		initialData: initial
	});
}

export function apiOptions(id: string, initial: API) {
	return queryOptions({
		queryKey: keys.apis.one(id),
		queryFn: async () => (await apisApi.get(id)).api,
		initialData: initial
	});
}

export function apiApplicationsOptions(id: string) {
	return queryOptions({
		queryKey: keys.apis.applications(id),
		queryFn: async () => (await apisApi.applications(id)).applications
	});
}

export function apiLogsOptions(id: string) {
	return queryOptions({
		queryKey: keys.apis.logs(id),
		queryFn: async () => (await apisApi.logs(id)).logs
	});
}
