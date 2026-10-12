import { queryOptions } from '@tanstack/svelte-query';

import { flowsApi, type LoginFlow, type LoginStepSpec } from '$lib/api';
import { keys } from './keys';

export function loginFlowsOptions(initial: { flows: LoginFlow[]; step_kinds: LoginStepSpec[] }) {
	return queryOptions({
		queryKey: keys.flows.list,
		queryFn: () => flowsApi.list(),
		initialData: initial
	});
}

export function loginFlowChoicesOptions() {
	return queryOptions({
		queryKey: keys.flows.list,
		queryFn: () => flowsApi.list(),
		retry: false
	});
}
