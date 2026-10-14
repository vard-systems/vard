import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { adminApi } from '#lib/api/index.js';
import { setupRequired } from '#lib/server/api.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	const { state } = await adminApi.session(fetch).catch(() => ({ state: 'none' as const }));

	if (state === 'signed_in') redirect(307, resolve('admin/dashboard'));
	if (state === 'enroll') redirect(307, resolve('admin/mfa-setup'));

	if (state === 'none' && (await setupRequired(fetch))) {
		redirect(307, resolve('admin/new-super-admin'));
	}

	return { waitingForCode: state === 'mfa' };
};
