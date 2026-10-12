import { signIn, type LoginOptions } from '$lib/api';
import { requireUser } from '$lib/server/session';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, fetch, url, setHeaders }) => {
	setHeaders({ 'cache-control': 'private, no-store' });

	const user = await requireUser(cookies, fetch, url);

	// A page that hides a control because one request failed is worse than one
	// that offers it and is refused, so this answers null rather than throwing.
	const [login, timezone] = await Promise.all([
		signIn
			.loginOptions('', fetch)
			.then((body): LoginOptions | null => body.login)
			.catch(() => null),
		signIn
			.organization(fetch)
			.then((body) => body.organization.timezone)
			.catch(() => 'UTC')
	]);

	return { user, login, timezone };
};
