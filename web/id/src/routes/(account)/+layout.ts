import { signIn, type LoginOptions } from '#lib/api/index.js';
import { requireUser } from '#lib/session.js';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ fetch, url }) => {
	const user = await requireUser(fetch, url);

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
