import { browser } from '$app/env';

import { COOKIES } from '#lib/brand.js';
import { currentTheme, type Theme } from '#lib/theme.js';
import { rememberChoice } from '#lib/utils/cookies.js';

export type { Theme };

function wantsMotion(): boolean {
	return !matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/** The theme the page is drawn in. The server writes the chosen one into
 *  <html> (hooks.server.ts), which the stylesheet keys off; this follows it
 *  in the browser and keeps a new choice in the reader's cookie. */
class ThemeState {
	current = $state<Theme>(browser ? currentTheme() : 'light');

	toggle() {
		this.set(this.current === 'dark' ? 'light' : 'dark');
	}

	set(theme: Theme) {
		const apply = () => {
			this.current = theme;
			document.documentElement.dataset.theme = theme;
			rememberChoice(COOKIES.theme, theme);
		};

		if (document.startViewTransition && wantsMotion()) {
			document.startViewTransition(apply);
			return;
		}
		apply();
	}
}

export const theme = new ThemeState();
