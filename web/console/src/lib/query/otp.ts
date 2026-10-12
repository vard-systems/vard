import { queryOptions } from '@tanstack/svelte-query';

import { otpApi, type OTPResponse } from '$lib/api';
import { keys } from './keys';

export function otpOptions(initial: OTPResponse) {
	return queryOptions({
		queryKey: keys.otp.settings,
		queryFn: () => otpApi.get(),
		initialData: initial
	});
}
