import type { HighlighterCore } from 'shiki/core';

let loading: Promise<HighlighterCore> | undefined;

function load(): Promise<HighlighterCore> {
	loading ??= (async () => {
		const [{ createCssVariablesTheme, createHighlighterCore }, { createJavaScriptRegexEngine }] =
			await Promise.all([import('shiki/core'), import('shiki/engine/javascript')]);

		return createHighlighterCore({
			themes: [createCssVariablesTheme({ name: 'panel', variablePrefix: '--syntax-' })],
			langs: [import('shiki/langs/json.mjs')],
			engine: createJavaScriptRegexEngine()
		});
	})();
	return loading;
}

async function highlight(code: string): Promise<string> {
	try {
		return (await load()).codeToHtml(code, { lang: 'json', theme: 'panel' });
	} catch {
		return '';
	}
}

export function highlighted(source: () => string): { readonly html: string } {
	let html = $state('');

	$effect(() => {
		const code = source();
		let current = true;

		const frame = requestAnimationFrame(() => {
			void highlight(code).then((result) => {
				if (current) html = result;
			});
		});

		return () => {
			current = false;
			cancelAnimationFrame(frame);
		};
	});

	return {
		get html() {
			return html;
		}
	};
}
