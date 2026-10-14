/**
 * Markdown to HTML at build time: GitHub Markdown plus
 *
 * - `## Heading {#id}` fixes an anchor so links survive rewording;
 * - `## POST /oauth2/token` headings render as endpoints;
 * - `> [!NOTE]`, `[!TIP]`, `[!WARNING]` callouts;
 * - code highlighted at build time in both themes, consecutive `title="…"` blocks becoming
 *   tabs;
 * - `{{name}}` and the other PLACEHOLDERS filled from #lib/brand.
 */
import { Marked, type Token, type Tokens } from 'marked';
import { createHighlighter, type Highlighter } from 'shiki';
import { href as docsHref } from '#lib/docs/links.js';
import { PLACEHOLDERS } from '#lib/brand.js';
import type { TocEntry } from '#lib/docs/types.js';

const LANGUAGES = ['java', 'go', 'python', 'bash', 'json', 'http', 'html'];
const ALIASES: Record<string, string> = {
	sh: 'bash',
	shell: 'bash',
	golang: 'go',
	py: 'python'
};
const METHODS = /^(GET|POST|PUT|PATCH|DELETE)\s+(\S+)$/;
const CALLOUTS: Record<string, string> = {
	NOTE: 'Note',
	TIP: 'Tip',
	IMPORTANT: 'Important',
	WARNING: 'Warning',
	CAUTION: 'Caution'
};

/** The copy button every block's head carries; the page wires them up.
    The icon is Remix Icon's file-copy-line, the set the console draws with. */
const COPY =
	`<button type="button" class="copy" data-copy aria-label="Copy the code">` +
	`<svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" aria-hidden="true"><path d="M7 6V3a1 1 0 0 1 1-1h12a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-3v3c0 .55-.45 1-1.01 1H4.01A1 1 0 0 1 3 21l.003-14c0-.552.45-1 1.007-1zM5.003 8 5 20h10V8zM9 6h8v10h2V4H9z"/></svg>` +
	`<span>Copy</span></button>`;

let highlighter: Promise<Highlighter> | undefined;

function getHighlighter(): Promise<Highlighter> {
	highlighter ??= createHighlighter({ themes: ['github-light', 'github-dark'], langs: LANGUAGES });
	return highlighter;
}

interface Rendered {
	html: string;
	toc: TocEntry[];
}

const escape = (text: string) =>
	text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

/** An anchor from a heading's words, the way GitHub makes one. */
function slugify(text: string): string {
	return text
		.toLowerCase()
		.replace(/<[^>]+>/g, '')
		.replace(/[^\p{L}\p{N}\s-]/gu, '')
		.trim()
		.replace(/\s+/g, '-');
}

/** A code block's info string: its language, and a title for a tab. */
function info(lang = ''): { language: string; title: string } {
	const [first = '', ...rest] = lang.trim().split(/\s+/);
	const title = /title="([^"]*)"/.exec(rest.join(' '))?.[1] ?? '';
	const language = ALIASES[first] ?? first;
	return { language: LANGUAGES.includes(language) ? language : 'text', title };
}

function fill(source: string): string {
	return source.replace(/\{\{(\w+)\}\}/g, (whole, key: string) => PLACEHOLDERS[key] ?? whole);
}

// The page and the search index both render every page while building;
// the second asks for what the first already made.
const renders = new Map<string, Promise<Rendered>>();

export function render(source: string): Promise<Rendered> {
	let rendered = renders.get(source);
	if (!rendered) {
		rendered = renderNow(source);
		renders.set(source, rendered);
	}
	return rendered;
}

async function renderNow(source: string): Promise<Rendered> {
	const shiki = await getHighlighter();
	const toc: TocEntry[] = [];
	const used = new Map<string, number>();
	const highlighted = new WeakMap<Tokens.Code, string>();
	let tabGroups = 0;

	const unique = (id: string) => {
		const count = used.get(id) ?? 0;
		used.set(id, count + 1);
		return count === 0 ? id : `${id}-${count}`;
	};

	const highlightedOf = (token: Tokens.Code) =>
		highlighted.get(token) ?? `<pre><code>${escape(token.text)}</code></pre>`;

	const codeBlock = (token: Tokens.Code) => {
		const { language, title } = info(token.lang);
		const label = title || (language === 'text' ? '' : language);
		return (
			`<div class="code-block">` +
			`<div class="code-head"><span>${escape(label)}</span>${COPY}</div>` +
			highlightedOf(token) +
			`</div>`
		);
	};

	const marked = new Marked({
		gfm: true,
		renderer: {
			heading({ tokens, depth, text }) {
				const pinned = /\s*\{#([\w-]+)\}\s*$/.exec(text);
				const plain = pinned ? text.slice(0, pinned.index) : text;
				const inner = this.parser.parseInline(tokens).replace(/\s*\{#[\w-]+\}\s*$/, '');
				const id = unique(pinned ? pinned[1] : slugify(plain));
				const endpoint = METHODS.exec(plain.trim());

				if (depth === 2 || depth === 3) {
					toc.push({
						id,
						depth,
						text: endpoint
							? endpoint[2]
							: plain.replace(/`/g, '').replace(/\[([^\]]*)\]\([^)]*\)/g, '$1'),
						method: endpoint?.[1]
					});
				}

				const content = endpoint
					? `<span class="method method-${endpoint[1].toLowerCase()}">${endpoint[1]}</span><code class="path">${escape(endpoint[2])}</code>`
					: inner;
				return `<h${depth} id="${id}"${endpoint ? ' class="endpoint"' : ''}>${content}<a class="anchor" href="#${id}" aria-label="Link to this section">#</a></h${depth}>\n`;
			},

			code(token) {
				return codeBlock(token);
			},

			blockquote({ tokens }) {
				const inner = this.parser.parse(tokens);
				const callout = /^<p>\[!(\w+)\]\s*/.exec(inner);
				if (callout && CALLOUTS[callout[1]]) {
					const kind = callout[1].toLowerCase();
					const body = inner.slice(callout[0].length).replace(/^<\/p>\s*/, '');
					const content = body.startsWith('<') ? body : `<p>${body}`;
					return `<div class="callout callout-${kind}" role="note"><p class="callout-title">${CALLOUTS[callout[1]]}</p>${content}</div>\n`;
				}
				return `<blockquote>${inner}</blockquote>\n`;
			},

			// HTML written into a page — the home page's cards — links the
			// way Markdown does, so its addresses get the base path too.
			html({ text }) {
				return text.replace(
					/href="(\/(?!\/)[^"]*)"/g,
					(_, path: string) => `href="${docsHref(path)}"`
				);
			},

			link({ href, title, tokens }) {
				const text = this.parser.parseInline(tokens);
				const titled = title ? ` title="${escape(title)}"` : '';
				if (/^https?:\/\//.test(href)) {
					return `<a href="${escape(href)}"${titled} target="_blank" rel="noopener noreferrer">${text}</a>`;
				}
				const local = href.startsWith('/') && !href.startsWith('//') ? docsHref(href) : href;
				return `<a href="${escape(local)}"${titled}>${text}</a>`;
			}
		}
	});

	const tokens = marked.lexer(fill(source));

	// Highlighting is asynchronous and rendering is not, so every block is
	// highlighted first and the renderer picks the result up.
	const blocks: Tokens.Code[] = [];
	marked.walkTokens(tokens, (token) => {
		if (token.type === 'code') blocks.push(token as Tokens.Code);
	});
	for (const block of blocks) {
		const { language } = info(block.lang);
		highlighted.set(
			block,
			shiki.codeToHtml(block.text, {
				lang: language,
				themes: { light: 'github-light', dark: 'github-dark' },
				defaultColor: false
			})
		);
	}

	// A run of titled code blocks, with nothing but blank lines between
	// them, is one example in several languages: tabs in one block's head.
	// They are radio buttons, so they work before any script has loaded;
	// the page keeps every block on the language the reader last chose.
	const grouped: Token[] = [];
	for (let i = 0; i < tokens.length; i++) {
		const run: Tokens.Code[] = [];
		let j = i;
		while (j < tokens.length) {
			const token = tokens[j];
			if (token.type === 'code' && info((token as Tokens.Code).lang).title) {
				run.push(token as Tokens.Code);
				j++;
			} else if (token.type === 'space' && run.length > 0) {
				j++;
			} else {
				break;
			}
		}

		if (run.length < 2) {
			grouped.push(tokens[i]);
			continue;
		}

		const name = `tabs-${++tabGroups}`;
		const titles = run.map((block) => escape(info(block.lang).title));
		const html =
			`<div class="code-block tabs">` +
			titles
				.map(
					(title, index) =>
						`<input type="radio" name="${name}" id="${name}-${index}" value="${title}" data-language${index === 0 ? ' checked' : ''}>`
				)
				.join('') +
			`<div class="code-head"><div class="tab-labels">` +
			titles.map((title, index) => `<label for="${name}-${index}">${title}</label>`).join('') +
			`</div>${COPY}</div>` +
			run.map((block) => `<div class="tab-panel">${highlightedOf(block)}</div>`).join('') +
			`</div>\n`;

		grouped.push({ type: 'html', raw: html, text: html, block: true, pre: false } as Tokens.HTML);
		i = j - 1;
	}

	const html = marked
		.parser(grouped)
		.replace(/<table>/g, '<div class="table-wrap"><table>')
		.replace(/<\/table>/g, '</table></div>');

	return { html, toc };
}
