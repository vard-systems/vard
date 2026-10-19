/**
 * Writes lib/icons.ts: the path of every Remix Icon the panel imports from it, read from the
 * `remixicon` package's SVGs. Only what is used is written, so the panel ships those and no more.
 *
 *   bun run icons          write it
 *   bun run icons:check    fail if it is behind what the panel imports (run by check)
 *
 * An icon is used by importing it, by its Remix Icon name in PascalCase with an `Ri` prefix
 * (`add-line` is `RiAddLine`), then running `bun run icons`.
 */
import { readdir, readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import prettier from 'prettier';

const source = new URL('../src/', import.meta.url);
const target = fileURLToPath(new URL('lib/icons.ts', source));
const library = fileURLToPath(new URL('../node_modules/remixicon/icons/', import.meta.url));

const importing = /import\s*\{([^}]*)\}\s*from\s*'#lib\/icons\.js'/g;

// Every SVG in the library, by the name the panel imports it as.
const available = new Map<string, string>();
for (const file of await readdir(library, { recursive: true })) {
	if (!file.endsWith('.svg')) continue;
	const stem = file.split(/[/\\]/).at(-1)!.slice(0, -'.svg'.length);
	const name = 'Ri' + stem.replace(/(^|-)([a-z0-9])/g, (_, __, char: string) => char.toUpperCase());
	available.set(name, library + file);
}

const used = new Set<string>();
for (const file of await readdir(fileURLToPath(source), { recursive: true })) {
	if (!/\.(svelte|ts)$/.test(file) || file.endsWith('icons.ts')) continue;
	const text = await readFile(fileURLToPath(new URL(file, source)), 'utf8');
	for (const [, names] of text.matchAll(importing)) {
		for (const name of names.split(',')) {
			const trimmed = name.trim().replace(/^type\s+/, '');
			if (trimmed && trimmed !== 'IconData') used.add(trimmed);
		}
	}
}

const unknown = [...used].filter((name) => !available.has(name));
if (unknown.length > 0) {
	console.error(`Remix Icon has no icon by these names: ${unknown.join(', ')}`);
	process.exit(1);
}

const entries: string[] = [];
for (const name of [...used].sort()) {
	const svg = await readFile(available.get(name)!, 'utf8');
	const paths = [...svg.matchAll(/<path d="([^"]+)"/g)].map(([, d]) => d);
	if (paths.length !== 1 || /<(circle|rect|polygon|ellipse|line|g)\b/.test(svg)) {
		console.error(`${name} is not a single path, which IconData cannot hold`);
		process.exit(1);
	}
	entries.push(`export const ${name} = '${paths[0]}' as IconData;`);
}

const output = `// Written by \`bun run icons\` from the remixicon package; do not edit by hand.

/** An icon: the path of a 24 × 24 Remix Icon, drawn by the Icon component. */
export type IconData = string & { readonly __brand: 'icon' };

${entries.join('\n')}
`;

const options = await prettier.resolveConfig(target);
const formatted = await prettier.format(output, { ...options, filepath: target });
const current = await readFile(target, 'utf8').catch(() => '');

if (current !== formatted) {
	if (process.argv.includes('--check')) {
		console.error(
			`lib/icons.ts does not match the icons the panel imports. Run \`bun run icons\`.`
		);
		process.exit(1);
	}
	await writeFile(target, formatted);
}
