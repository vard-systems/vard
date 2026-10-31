import { describe, expect, it } from 'vitest';
import { longEnough, newPasswordReady } from './password';

describe('longEnough', () => {
	it.each([
		['eleven ASCII characters', 'abcdefghijk', false],
		['twelve ASCII characters', 'abcdefghijkl', true],
		// The server counts runes: six emoji are six characters, though
		// String.length says twelve.
		['six emoji', '🔑'.repeat(6), false],
		['twelve emoji', '🔑'.repeat(12), true],
		['twelve accented letters', 'é'.repeat(12), true]
	])('%s', (_, password, expected) => expect(longEnough(password)).toBe(expected));
});

describe('newPasswordReady', () => {
	it.each([
		['long enough and typed twice', 'correct horse battery', 'correct horse battery', true],
		['typed differently', 'correct horse battery', 'correct house battery', false],
		['not confirmed yet', 'correct horse battery', '', false],
		['too short', 'short', 'short', false]
	])('%s', (_, password, confirm, expected) =>
		expect(newPasswordReady(password, confirm)).toBe(expected)
	);
});
