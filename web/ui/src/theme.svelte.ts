export type ThemeName = 'light' | 'dark';

/** The attribute the stylesheet keys off (styles/tokens.css). */
const ATTRIBUTE = 'data-theme';
const ONE_YEAR = 60 * 60 * 24 * 365;

const browser = typeof document !== 'undefined';

function isTheme(value: unknown): value is ThemeName {
	return value === 'light' || value === 'dark';
}

function wantsMotion(): boolean {
	return browser && !window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * An app's light or dark theme. It is kept in the app's own cookie, which the app's
 * static/theme.js reads before the first paint, so a page is never drawn in the wrong one.
 */
export class Theme {
	current = $state<ThemeName>('light');

	constructor(private readonly cookie: string) {
		this.current = this.initial();
	}

	toggle() {
		this.set(this.current === 'dark' ? 'light' : 'dark');
	}

	set(theme: ThemeName) {
		if (!browser) {
			this.current = theme;
			return;
		}

		const apply = () => {
			this.current = theme;
			document.documentElement.setAttribute(ATTRIBUTE, theme);
			document.cookie = `${this.cookie}=${theme}; path=/; max-age=${ONE_YEAR}; samesite=lax`;
		};

		if (document.startViewTransition && wantsMotion()) {
			document.startViewTransition(apply);
			return;
		}

		apply();
	}

	private initial(): ThemeName {
		if (!browser) return 'light';

		const attribute = document.documentElement.getAttribute(ATTRIBUTE);
		if (isTheme(attribute)) return attribute;

		const match = document.cookie.match(new RegExp(`(?:^|; )${this.cookie}=([^;]*)`));
		if (isTheme(match?.[1])) return match[1];

		return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
	}
}
