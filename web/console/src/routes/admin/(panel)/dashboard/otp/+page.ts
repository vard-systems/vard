import type { OTPResponse } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
	requirePermission((await parent()).admin, 'super_admin');

	return { otp: await apiGet<OTPResponse>('/admin/otp', fetch) };
};
