import { otpApi } from '#lib/api/index.js';
import { fromApi, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	requirePermission((await parent()).admin, 'super_admin');

	return { otp: await fromApi(otpApi.get(fetch)) };
};
