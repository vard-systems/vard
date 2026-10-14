import { pages } from '#lib/server/content.js';
import { render } from '#lib/server/markdown.js';
import type { SearchEntry } from '#lib/docs/types.js';

// What the search box looks through: every page, and every heading on one —
// which, on a reference page, is every endpoint. Written once while building,
// and fetched the first time somebody opens search.
export const prerender = true;

export async function GET() {
	const entries: SearchEntry[] = [];

	for (const page of pages) {
		const href = `/${page.slug}`;
		entries.push({ title: page.title, page: page.title, group: page.section, href });

		const { toc } = await render(page.source);
		for (const heading of toc) {
			entries.push({
				title: heading.text,
				page: page.title,
				group: page.section,
				href: `${href}#${heading.id}`,
				method: heading.method
			});
		}
	}

	return Response.json(entries);
}
