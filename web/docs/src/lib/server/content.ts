/**
 * Every Markdown file under src/content, read at build time. A file's path is its address and
 * its directory places it in the sidebar; guides choose their section with `section:` so
 * regrouping never moves an address. content/reference is written by `make docs`.
 */
import type { NavSection } from '#lib/docs/types.js';

export interface Page {
	/** The address without the leading slash: "" for the home page. */
	slug: string;
	/** Where the file is, relative to the app, for "edit this page". */
	file: string;
	/** The directory's group, which says how the page is shown. */
	group: string;
	/** The sidebar section it is listed under, and its eyebrow. */
	section: string;
	title: string;
	/** What the sidebar calls it: `nav:` in the front matter, or the title. */
	nav: string;
	description: string;
	order: number;
	generated: boolean;
	/** A Remix Icon name from the front matter, for the sidebar row. */
	icon: string;
	source: string;
}

/**
 * Sidebar groups in order, with their label and whether they show as rows or one branch.
 * Unlisted directories are not published.
 */
const GROUPS: { directory: string; title: string; section: string; branch?: string }[] = [
	{ directory: '', title: 'Overview', section: 'Get started' },
	{ directory: 'start', title: 'Getting started', section: 'Get started' },
	// Each guide names its section; this is the one it falls back to.
	{ directory: 'guides', title: 'Guides', section: 'Tools' },
	{
		directory: 'reference/public',
		title: 'Public API',
		section: 'API reference',
		branch: 'global'
	},
	{ directory: 'reference', title: 'Reference', section: 'API reference' },
	{ directory: 'reference/admin', title: 'Admin API', section: 'API reference', branch: 'server' }
];

/**
 * The sidebar's sections, in order: what a reader is doing, from their first
 * sign-in to the full reference. A page's `section:` must be one of these.
 */
const SECTIONS = [
	'Get started',
	'Sign in',
	'Protect APIs',
	'Manage users',
	'Tools',
	'API reference'
];

const files = import.meta.glob('/src/content/**/*.md', {
	query: '?raw',
	import: 'default',
	eager: true
}) as Record<string, string>;

/** Reads the front matter: `key: value` lines, values plain or JSON-quoted. */
function frontMatter(raw: string): { data: Record<string, string>; body: string } {
	const match = /^---\n([\s\S]*?)\n---\n?/.exec(raw);
	if (!match) return { data: {}, body: raw };

	const data: Record<string, string> = {};
	for (const line of match[1].split('\n')) {
		const colon = line.indexOf(':');
		if (colon < 0) continue;
		const key = line.slice(0, colon).trim();
		let value = line.slice(colon + 1).trim();
		if (value.startsWith('"')) {
			try {
				value = JSON.parse(value);
			} catch {
				// A value that only looks quoted is kept as it was written.
			}
		} else if (value.length > 1 && value.startsWith("'") && value.endsWith("'")) {
			// YAML's single quotes, which Prettier prefers: '' is a quote.
			value = value.slice(1, -1).replace(/''/g, "'");
		}
		data[key] = value;
	}

	return { data, body: raw.slice(match[0].length) };
}

function load(): Page[] {
	const pages: Page[] = [];

	for (const [path, raw] of Object.entries(files)) {
		const relative = path.replace(/^\/src\/content\//, '').replace(/\.md$/, '');
		const parts = relative.split('/');
		const name = parts.pop() as string;
		const directory = parts.join('/');

		const group = GROUPS.find((g) => g.directory === directory);
		if (!group) continue;

		const { data, body } = frontMatter(raw);
		const slug = name === 'index' ? directory : relative;

		const section = data.section || group.section;
		if (!SECTIONS.includes(section)) {
			throw new Error(`${path}: section "${section}" is not one of ${SECTIONS.join(', ')}`);
		}

		pages.push({
			slug,
			file: path.slice(1),
			group: group.title,
			section,
			title: data.title || name,
			nav: data.nav || data.title || name,
			description: data.description ?? '',
			order: Number(data.order ?? 100),
			generated: data.generated === 'true',
			icon: data.icon ?? 'file-list-3',
			source: body
		});
	}

	const section = (page: Page) => SECTIONS.indexOf(page.section);
	const group = (page: Page) => GROUPS.findIndex((g) => g.title === page.group);
	return pages.sort(
		(a, b) =>
			section(a) - section(b) ||
			group(a) - group(b) ||
			a.order - b.order ||
			a.title.localeCompare(b.title)
	);
}

/** Every page, in reading order: the sidebar's, and prev/next's. */
export const pages: Page[] = load();

export function pageAt(slug: string): Page | undefined {
	return pages.find((page) => page.slug === slug);
}

/** The sidebar: labelled sections of rows and branches, in reading order. */
export function navigation(): NavSection[] {
	const sections: NavSection[] = [];

	for (const page of pages) {
		const group = GROUPS.find((g) => g.title === page.group)!;
		let section = sections.find((s) => s.label === page.section);
		if (!section) {
			section = { label: page.section, items: [] };
			sections.push(section);
		}

		const link = { title: page.nav, href: `/${page.slug}`, icon: page.icon };
		if (!group.branch) {
			section.items.push({ kind: 'link', ...link });
			continue;
		}

		let branch = section.items.find(
			(item) => item.kind === 'branch' && item.id === group.directory
		);
		if (!branch) {
			branch = {
				kind: 'branch',
				id: group.directory,
				title: group.title,
				icon: group.branch,
				pages: []
			};
			section.items.push(branch);
		}
		// A branch's own index is its overview, under the branch's name.
		const isIndex = page.slug === group.directory;
		if (branch.kind === 'branch')
			branch.pages.push(isIndex ? { ...link, title: 'Overview' } : link);
	}

	return sections;
}
