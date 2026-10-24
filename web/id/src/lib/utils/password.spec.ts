import { describe, expect, it } from 'vitest';
import { longEnough, newPasswordReady } from './password';

describe('longEnough', () => {
	it.each([
		['seven ASCII characters', 'abcdefg', false],
		['eight ASCII characters', 'abcdefgh', true],
		// The server counts runes: four emoji are four characters, though
		// String.length says eight.
		['four emoji', '🔑🔑🔑🔑', false],
		['eight emoji', '🔑🔑🔑🔑🔑🔑🔑🔑', true],
		['eight accented letters', 'éééééééé', true]
	])('%s', (_, password, expected) => expect(longEnough(password)).toBe(expected));
});

describe('newPasswordReady', () => {
	it.each([
		['long enough and typed twice', 'correct horse', 'correct horse', true],
		['typed differently', 'correct horse', 'correct house', false],
		['not confirmed yet', 'correct horse', '', false],
		['too short', 'short', 'short', false]
	])('%s', (_, password, confirm, expected) =>
		expect(newPasswordReady(password, confirm)).toBe(expected)
	);
});
