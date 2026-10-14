import { error } from '@sveltejs/kit';
import { pageAt, pages } from '#lib/server/content.js';
import { render } from '#lib/server/markdown.js';
import type { EntryGenerator, PageServerLoad } from './$types';

// Every page, so the build renders the ones no link reaches as well.
export const entries: EntryGenerator = () => pages.map((page) => ({ slug: page.slug }));

export const load: PageServerLoad = async ({ params }) => {
	const slug = params.slug.replace(/\/$/, '');
	const page = pageAt(slug);
	if (!page) error(404, 'There is no page here.');

	const { html, toc } = await render(page.source);
	const index = pages.indexOf(page);
	const neighbour = (i: number) =>
		pages[i] ? { title: pages[i].title, href: `/${pages[i].slug}` } : null;

	return {
		title: page.title,
		description: page.description,
		group: page.section,
		generated: page.generated,
		file: page.file,
		html,
		toc,
		previous: neighbour(index - 1),
		next: neighbour(index + 1)
	};
};
