import { queryOptions } from '@tanstack/svelte-query';

import { adminApi, mfaApi } from '$lib/api';
import { keys } from './keys';

export function profileSessionsOptions() {
	return queryOptions({
		queryKey: keys.profile.sessions,
		queryFn: async () => (await adminApi.profileSessions()).sessions
	});
}

export function mfaStatusOptions() {
	return queryOptions({
		queryKey: keys.profile.mfa,
		queryFn: async () => (await mfaApi.status()).mfa
	});
}
