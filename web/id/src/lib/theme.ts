export type Theme = 'light' | 'dark';

export function isTheme(value: unknown): value is Theme {
	return value === 'light' || value === 'dark';
}

/** The theme the page is drawn in now: the one chosen, else the system's. */
export function currentTheme(): Theme {
	const chosen = document.documentElement.dataset.theme;
	if (isTheme(chosen)) return chosen;
	return matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}
