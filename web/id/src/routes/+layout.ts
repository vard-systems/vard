import { redirect } from '@sveltejs/kit';
import { signIn, type PublicLanguage } from '#lib/api/index.js';
import { COOKIES, LANGUAGE_DEPENDENCY } from '#lib/brand.js';
import { BASE, chooseLanguage, type Messages } from '#lib/i18n/index.js';
import { safeNext } from '#lib/utils/next.js';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ cookies, depends, fetch, locals, request, url }) => {
	depends(LANGUAGE_DEPENDENCY);

	const asked = url.searchParams.get('lang') ?? undefined;

	const offered = await signIn
		.languages(fetch)
		.catch(() => ({ languages: [] as PublicLanguage[], default: BASE }));

	const language = chooseLanguage(
		asked ?? cookies.get(COOKIES.language),
		request.headers.get('accept-language'),
		offered.languages.map((one) => one.code),
		offered.default
	);

	if (asked !== undefined) {
		cookies.set(COOKIES.language, language, {
			path: '/',
			httpOnly: false,
			sameSite: 'lax',
			maxAge: 60 * 60 * 24 * 365
		});

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

	locals.language = language;

	return { language, languages: offered.languages, messages };
};
