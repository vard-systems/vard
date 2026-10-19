import { error } from '@sveltejs/kit';
import {
	ApiError,
	signIn,
	type LoginOptions,
	type Organization,
	type SignInRequest,
	type SocialProvider,
	type SSOConnection
} from '#lib/api/index.js';
import type { LayoutLoad } from './$types';

export const load: LayoutLoad = async ({ url, fetch }) => {
	const request = url.searchParams.get('request');

	const [organization, socialProviders, sso, login] = await Promise.all([
		signIn
			.organization(fetch)
			.then(({ organization }): Organization | null => organization)
			.catch(() => null),
		signIn
			.socialProviders(fetch)
			.then(({ providers }): SocialProvider[] => providers)
			.catch((): SocialProvider[] => []),
		signIn
			.ssoConnections(fetch)
			.catch(() => ({ connections: [] as SSOConnection[], available: false })),
		signIn
			.loginOptions(request ?? '', fetch)
			.then(({ login }): LoginOptions => login)
			.catch((): LoginOptions => fallback)
	]);

	let signInRequest: SignInRequest | null = null;
	let expired = false;

	if (request) {
		try {
			signInRequest = await signIn.request(request, fetch);
		} catch (err) {
			if (err instanceof ApiError && (err.status === 410 || err.status === 404)) {
				expired = true;
			} else if (err instanceof ApiError && err.status === 0) {
				error(503, err.message);
			} else {
				throw err;
			}
		}
	}

	return {
		request,
		signInRequest,
		expired,
		organization,
		socialProviders,
		ssoConnections: sso.connections,
		ssoAvailable: sso.available,
		login
	};
};

/** What these pages offer when the server cannot say: the sign-in they have
    always offered. A page that hides its way to a new account because one
    request failed would be worse than one that offers a way the server then
    refuses. */
const fallback: LoginOptions = {
	steps: ['identifier', 'password', 'social'],
	allow_sign_in: true,
	allow_registration: true,
	allow_password_reset: true,
	allow_remember_me: true,
	allow_email_change: false
};
