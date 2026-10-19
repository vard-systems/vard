import { redirect } from '@sveltejs/kit';
import { signIn, type PublicLanguage } from '#lib/api/index.js';
import { COOKIES, LANGUAGE_DEPENDENCY } from '#lib/brand.js';
import { BASE, chooseLanguage, type Messages } from '#lib/i18n/index.js';
import { rememberLanguage } from '#lib/i18n/switch.js';
import { readCookie } from '#lib/utils/cookies.js';
import { safeNext } from '#lib/utils/next.js';
import type { LayoutLoad } from './$types';

// A single-page app: the Go server sends one index.html for every route, and
// everything is drawn in the browser from the API's answers.
export const ssr = false;
export const prerender = false;

export const load: LayoutLoad = async ({ depends, fetch, url }) => {
	depends(LANGUAGE_DEPENDENCY);

	const asked = url.searchParams.get('lang') ?? undefined;

	const offered = await signIn
		.languages(fetch)
		.catch(() => ({ languages: [] as PublicLanguage[], default: BASE }));

	const language = chooseLanguage(
		asked ?? readCookie(COOKIES.language),
		navigator.languages.join(','),
		offered.languages.map((one) => one.code),
		offered.default
	);

	if (asked !== undefined) {
		rememberLanguage(language);

		const clean = new URL(url);
		clean.searchParams.delete('lang');

		// Through safeNext, not the raw pathname: this load runs for a
		// path no route matches too, and a path that begins "//" would make
		// the Location protocol-relative — an open redirect on a sign-in
		// page, without an account or a sign-in.
		redirect(303, safeNext(`${clean.pathname}${clean.search}`));
	}

	const messages: Messages = offered.languages.length
		? await signIn
				.languageText(language, fetch)
				.then((text) => text.messages)
				.catch(() => ({}))
		: {};

	return { language, languages: offered.languages, messages };
};
