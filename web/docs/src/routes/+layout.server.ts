import { navigation } from '#lib/server/content.js';

// Every page is Markdown that does not change after a build, so every page is
// rendered while building and served as a file.
export const prerender = true;

export function load() {
	return { navigation: navigation() };
}
