import { describe, expect, it } from 'vitest';
import { longEnoughForAdmin } from './password';

describe('longEnoughForAdmin', () => {
	it.each([
		['nine ASCII characters', 'abcdefghi', false],
		['ten ASCII characters', 'abcdefghij', true],
		// The server counts runes: five emoji are five characters, though
		// String.length says ten.
		['five emoji', '🔑🔑🔑🔑🔑', false],
		['ten emoji', '🔑🔑🔑🔑🔑🔑🔑🔑🔑🔑', true]
	])('%s', (_, password, expected) => expect(longEnoughForAdmin(password)).toBe(expected));
});
