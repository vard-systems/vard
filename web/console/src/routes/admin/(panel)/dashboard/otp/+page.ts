import type { OTPResponse } from '#lib/api/index.js';
import { apiGet, requirePermission } from '#lib/server/api.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch, parent }) => {
	requirePermission((await parent()).admin, 'super_admin');

	return { otp: await apiGet<OTPResponse>('/admin/otp', fetch) };
};
