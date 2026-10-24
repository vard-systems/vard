import axe from 'axe-core';
import { mount, unmount, type Component } from 'svelte';
import { expect } from 'vitest';

// jsdom has no matchMedia, which the theme reads as soon as its module loads; a test page
// prefers nothing. Imported before the components, so it is there when they load.
window.matchMedia ??= (query: string) =>
	({
		matches: false,
		media: query,
		onchange: null,
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => false
	}) as unknown as MediaQueryList;

/**
 * Mounts a component in the test's document and fails on anything axe finds. Colour contrast
 * is left out: jsdom lays nothing out, so it cannot be measured here.
 */
export async function expectAccessible<Props extends Record<string, unknown>>(
	component: Component<Props>,
	props: Props
): Promise<void> {
	const target = document.createElement('main');
	document.body.append(target);
	const mounted = mount(component, { target, props });

	try {
		const { violations } = await axe.run(target, {
			rules: { 'color-contrast': { enabled: false } }
		});
		const found = violations.map(
			(violation) =>
				`${violation.id}: ${violation.help} (${violation.nodes.map((node) => node.html).join(', ')})`
		);
		expect(found, `axe found:\n${found.join('\n')}`).toEqual([]);
	} finally {
		await unmount(mounted);
		target.remove();
	}
}
