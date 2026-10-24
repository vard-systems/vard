import { nextUnlessSignedIn } from '#lib/session.js';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ url, fetch }) => ({
	next: await nextUnlessSignedIn(fetch, url)
});
