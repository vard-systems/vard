import { ONE_YEAR } from '#lib/constants.js';

/** Keeps a choice of the reader's (language, theme) for a year, on this host only. */
export function rememberChoice(name: string, value: string): void {
	document.cookie = `${name}=${encodeURIComponent(value)}; path=/; max-age=${ONE_YEAR}; samesite=lax`;
}
