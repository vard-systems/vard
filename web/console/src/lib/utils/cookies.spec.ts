import type { Cookies } from '@sveltejs/kit';
import { describe, expect, it } from 'vitest';
import { choiceName, readChoice, setChoice } from './cookies';

function jar(initial: Record<string, string> = {}) {
	const values = new Map(Object.entries(initial));
	const options = new Map<string, unknown>();
	const cookies = {
		get: (name: string) => values.get(name),
		set: (name: string, value: string, opts: unknown) => {
			values.set(name, value);
			options.set(name, opts);
		}
	} as unknown as Cookies;
	return { cookies, values, options };
}

const http = new URL('http://localhost:5173/');
const https = new URL('https://id.example.com/');

describe('choiceName', () => {
	it.each([
		['plain http keeps the bare name', false, 'app-theme'],
		['https takes the __Host- prefix', true, '__Host-app-theme']
	])('%s', (_, secure, want) => {
		expect(choiceName('app-theme', secure)).toBe(want);
	});
});

describe('readChoice', () => {
	it('reads the name for the page’s scheme alone', () => {
		const { cookies } = jar({ 'app-theme': 'light', '__Host-app-theme': 'dark' });
		expect(readChoice(cookies, 'app-theme', http)).toBe('light');
		expect(readChoice(cookies, 'app-theme', https)).toBe('dark');
	});

	it('does not take another app’s cookie that ends the same way', () => {
		const { cookies } = jar({ 'app-account-theme': 'dark' });
		expect(readChoice(cookies, 'app-theme', http)).toBeUndefined();
	});

	it('does not take a bare cookie over https, where a sibling could have planted it', () => {
		const { cookies } = jar({ 'app-theme': 'dark' });
		expect(readChoice(cookies, 'app-theme', https)).toBeUndefined();
	});
});

describe('setChoice', () => {
	it('writes a Secure __Host- cookie over https', () => {
		const { cookies, values, options } = jar();
		setChoice(cookies, 'app-language', 'de', https);
		expect(values.get('__Host-app-language')).toBe('de');
		expect(options.get('__Host-app-language')).toMatchObject({ path: '/', secure: true });
	});

	it('writes the bare name over plain http', () => {
		const { cookies, options } = jar();
		setChoice(cookies, 'app-language', 'de', http);
		expect(options.get('app-language')).toMatchObject({ path: '/', secure: false });
	});
});
