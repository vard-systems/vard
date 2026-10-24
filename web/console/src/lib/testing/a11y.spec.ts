// @vitest-environment jsdom
import { describe, expect, it } from 'vitest';
import { expectAccessible } from './a11y';
import Unlabelled from './Unlabelled.svelte';

describe('the accessibility check', () => {
	it('fails on a field and a picture with no name', async () => {
		await expect(expectAccessible(Unlabelled, {})).rejects.toThrow(/image-alt|label/);
	});
});
