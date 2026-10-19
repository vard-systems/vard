import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import { setupRequired } from '#lib/api/load.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	if (!(await setupRequired(fetch))) {
		redirect(307, resolve('/login'));
	}

	return {};
};
