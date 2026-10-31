import { readdirSync, readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

// An icon is a class of the Remix Icon font, which draws nothing for a name it
// does not have: every one the app writes must be in the stylesheet it loads.
const stylesheet = readFileSync(
	new URL('../../node_modules/remixicon/fonts/remixicon.css', import.meta.url),
	'utf8'
);
const source = new URL('..', import.meta.url);

const used = new Map<string, string>();
for (const file of readdirSync(source, { recursive: true, encoding: 'utf8' })) {
	if (!/\.(svelte|ts)$/.test(file) || file.endsWith('icons.spec.ts')) continue;
	const text = readFileSync(new URL(file, source), 'utf8');
	for (const [name] of text.matchAll(/\bri-[a-z0-9]+(?:-[a-z0-9]+)*/g)) {
		if (!used.has(name)) used.set(name, file);
	}
}

describe('Remix Icon classes', () => {
	it('the app uses some', () => expect(used.size).toBeGreaterThan(0));

	for (const [name, file] of used) {
		it(`${name} (${file}) is in the font`, () => expect(stylesheet).toContain(`.${name}:before`));
	}
});
