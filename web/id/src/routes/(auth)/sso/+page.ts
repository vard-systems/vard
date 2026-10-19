import { redirect } from '@sveltejs/kit';
import { currentUser } from '#lib/session.js';
import { safeNext } from '#lib/utils/next.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => {
	const next = safeNext(url.searchParams.get('next'));

	if (!url.searchParams.get('request') && (await currentUser(fetch))) {
		redirect(303, next);
	}

	return { next };
};
