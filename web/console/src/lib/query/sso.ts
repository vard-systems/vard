import { queryOptions } from '@tanstack/svelte-query';

import { ssoApi, type SSOConnection } from '#lib/api/index.js';
import { keys } from './keys';

export function ssoOptions(initial: { connections: SSOConnection[] }) {
	return queryOptions({
		queryKey: keys.sso.list,
		queryFn: () => ssoApi.list(),
		initialData: initial
	});
}

/** A saved connection's domains and whether each is proved. */
export function ssoDomainsOptions(id: string) {
	return queryOptions({
		queryKey: keys.sso.domains(id),
		queryFn: async () => (await ssoApi.domains(id)).domains
	});
}
