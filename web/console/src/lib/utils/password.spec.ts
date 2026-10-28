import { describe, expect, it } from 'vitest';
import { longEnoughForAdmin } from './password';

describe('longEnoughForAdmin', () => {
	it.each([
		['thirteen ASCII characters', 'abcdefghijklm', false],
		['fourteen ASCII characters', 'abcdefghijklmn', true],
		// The server counts runes: seven emoji are seven characters, though
		// String.length says fourteen.
		['seven emoji', '🔑'.repeat(7), false],
		['fourteen emoji', '🔑'.repeat(14), true]
	])('%s', (_, password, expected) => expect(longEnoughForAdmin(password)).toBe(expected));
});
