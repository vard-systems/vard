import { redirect } from '@sveltejs/kit';
import { currentUser } from '#lib/server/session.js';
import { safeNext } from '#lib/utils/next.js';
import type { PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ url, cookies, fetch }) => {
	const next = safeNext(url.searchParams.get('next'));

	if (!url.searchParams.get('request') && (await currentUser(cookies, fetch))) {
		redirect(303, next);
	}

	return { next };
};
