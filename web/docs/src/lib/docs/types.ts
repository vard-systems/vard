/** A page in the sidebar. */
export interface NavPage {
	title: string;
	href: string;
	/** A Remix Icon name, as #lib/docs/icons.js knows them. */
	icon: string;
}

/** A row of the sidebar: a page of its own, or a branch that opens onto
    several — the console's two kinds of row. */
export type NavItem =
	| ({ kind: 'link' } & NavPage)
	| { kind: 'branch'; id: string; title: string; icon: string; pages: NavPage[] };

/** A labelled group of rows, like the console's Overview and Manage. */
export interface NavSection {
	label: string;
	items: NavItem[];
}

/** A heading the page's table of contents lists. */
export interface TocEntry {
	id: string;
	text: string;
	depth: number;
	/** The HTTP method, for a heading that names an endpoint. */
	method?: string;
}

/** One thing the search box can find: a page, or a heading on one. */
export interface SearchEntry {
	title: string;
	page: string;
	group: string;
	href: string;
	method?: string;
}
