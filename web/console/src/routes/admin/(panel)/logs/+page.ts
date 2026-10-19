import { redirect } from '@sveltejs/kit';
import { resolve } from '$app/paths';
import type { PageLoad } from './$types';

/** The logs moved into the dashboard, beside the activity they detail. An
    old link or bookmark still lands there. */
export const load: PageLoad = () => {
	redirect(308, resolve('/admin/(panel)/dashboard/logs'));
};
