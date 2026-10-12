import { browser } from '$app/environment';

import { COOKIES } from '$lib/brand';

export type Theme = 'light' | 'dark';

/** The theme is kept in a cookie rather than localStorage so the server can
 *  read it and send the page already themed. The attribute below is what the
 *  stylesheet keys off. */
const COOKIE = COOKIES.theme;
const ATTRIBUTE = 'data-theme';
const ONE_YEAR = 60 * 60 * 24 * 365;

function isTheme(value: unknown): value is Theme {
	return value === 'light' || value === 'dark';
}

function fromCookie(): Theme | null {
	const match = document.cookie.match(new RegExp(`(?:^|; )${COOKIE}=([^;]*)`));
	const value = match?.[1];

	return isTheme(value) ? value : null;
}

function initial(): Theme {
	if (!browser) return 'light';

	const attribute = document.documentElement.getAttribute(ATTRIBUTE);
	if (isTheme(attribute)) return attribute;

	return (
		fromCookie() ?? (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light')
	);
}

function wantsMotion(): boolean {
	return browser && !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

class ThemeState {
	current = $state<Theme>(initial());

	toggle() {
		this.set(this.current === 'dark' ? 'light' : 'dark');
	}

	set(theme: Theme) {
		if (!browser) {
			this.current = theme;
			return;
		}

		const apply = () => {
			this.current = theme;
			document.documentElement.setAttribute(ATTRIBUTE, theme);
			document.cookie = `${COOKIE}=${theme}; path=/; max-age=${ONE_YEAR}; samesite=lax`;
		};

		if (document.startViewTransition && wantsMotion()) {
			document.startViewTransition(apply);
			return;
		}

		apply();
	}
}

export const theme = new ThemeState();
