import { queryOptions } from '@tanstack/svelte-query';

import { organizationApi, type Organization } from '$lib/api';
import { keys } from './keys';

export function organizationOptions(initial: { organization: Organization }) {
	return queryOptions({
		queryKey: keys.organization.settings,
		queryFn: () => organizationApi.get(),
		initialData: initial
	});
}
