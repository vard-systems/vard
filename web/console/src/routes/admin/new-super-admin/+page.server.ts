import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { setupRequired } from '$lib/server/api';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ fetch }) => {
	if (!(await setupRequired(fetch))) {
		redirect(307, resolve('/admin/login'));
	}

	return {};
};
