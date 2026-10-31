/**
 * Writes lib/styles/tokens.css and lib/styles/fonts.css from lib/theme.
 *
 *   bun run theme          write them
 *   bun run theme:check    fail if either is behind the theme (run by check)
 */
import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import prettier from 'prettier';
import { renderFonts, renderTokens } from '../src/lib/theme/render';
import { theme } from '../src/lib/theme/theme';

const styles = new URL('../src/lib/styles/', import.meta.url);

const outputs = [
	{ file: new URL('tokens.css', styles), css: renderTokens(theme) },
	{ file: new URL('fonts.css', styles), css: renderFonts(theme) }
];

const checking = process.argv.includes('--check');
const stale: string[] = [];

for (const { file, css } of outputs) {
	const path = fileURLToPath(file);
	const options = await prettier.resolveConfig(path);
	const formatted = await prettier.format(css, { ...options, filepath: path });

	const current = await readFile(path, 'utf8').catch(() => '');
	if (current === formatted) continue;

	if (checking) stale.push(path);
	else await writeFile(path, formatted);
}

if (stale.length > 0) {
	console.error(
		`These files do not match lib/theme/theme.ts — the theme changed, or they were edited by hand:\n  ${stale.join('\n  ')}\nRun \`bun run theme\`.`
	);
	process.exit(1);
}
