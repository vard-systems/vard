import type { Theme, Tokens } from './types';

const HEADER = (source: string) =>
	`/* Generated from lib/theme/theme.ts by \`bun run theme\` — do not edit.
   ${source} */\n\n`;

const DARK_CHOSEN = ":root[data-theme='dark']";
const DARK_SYSTEM = ":root:not([data-theme='light'])";

function declarations(tokens: Partial<Tokens>, indent = '\t'): string {
	return Object.entries(tokens)
		.filter((entry): entry is [string, string] => entry[1] !== undefined)
		.map(([name, value]) => `${indent}--${name}: ${value};`)
		.join('\n');
}

function block(selector: string, body: string): string {
	return `${selector} {\n${body}\n}`;
}

function scaleTokens(theme: Theme): Tokens {
	return Object.fromEntries(
		theme.scales.flatMap(({ prefix, tokens }) =>
			Object.entries(tokens).map(([name, value]) => [prefix ? `${prefix}-${name}` : name, value])
		)
	);
}

export function renderTokens(theme: Theme): string {
	const { light, dark } = theme.colors;

	const root = block(
		':root',
		[declarations(scaleTokens(theme)), declarations(light), `\tcolor-scheme: light;`].join('\n\n')
	);

	const responsive = theme.responsive.map(({ minWidth, tokens }) =>
		block(`@media (min-width: ${minWidth})`, block('\t:root', declarations(tokens, '\t\t')))
	);

	const layers = theme.layers.sets.map(({ prefix, selectors }) =>
		block(
			selectors.join(',\n'),
			Object.entries(theme.layers.swapped)
				.map(([page, name]) => `\t--${page}: var(--${prefix}-${name});`)
				.join('\n')
		)
	);

	const darkBody = [declarations(dark), '\tcolor-scheme: dark;'].join('\n\n');
	const darkChosen = block(DARK_CHOSEN, darkBody);
	const darkSystem = block(
		'@media (prefers-color-scheme: dark)',
		block(
			`\t${DARK_SYSTEM}`,
			darkBody
				.split('\n')
				.map((line) => (line ? `\t${line}` : line))
				.join('\n')
		)
	);

	return (
		HEADER('The values every stylesheet and component refers to, light and dark.') +
		[root, ...responsive, ...layers, darkChosen, darkSystem].join('\n\n') +
		'\n'
	);
}

export function renderFonts(theme: Theme): string {
	const { baseUrl, subsets, faces } = theme.fonts;

	const rules = faces.flatMap((face) =>
		face.weights.flatMap((weight) =>
			face.subsets.map((subset) =>
				block(
					'@font-face',
					[
						`\tfont-family: '${face.family}';`,
						`\tfont-style: ${face.style};`,
						`\tfont-weight: ${weight};`,
						'\tfont-display: swap;',
						`\tsrc: url('${baseUrl}${face.file(subset, weight)}') format('woff2');`,
						`\tunicode-range: ${subsets[subset]};`
					].join('\n')
				)
			)
		)
	);

	return HEADER('The typefaces the console bundles.') + rules.join('\n\n') + '\n';
}
