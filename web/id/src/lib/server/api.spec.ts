import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/env/private', () => ({ API_URL: 'http://api.test' }));

const { isApiPath } = await import('./api');

describe('isApiPath', () => {
	const tests: { name: string; path: string; want: boolean }[] = [
		{ name: 'a provider endpoint', path: '/oauth2/token', want: true },
		{ name: 'the discovery document', path: '/.well-known/openid-configuration', want: true },
		{ name: 'an account endpoint', path: '/api/v1/account/me', want: true },
		{ name: 'a page', path: '/login', want: false },
		{ name: 'a page that only starts like an endpoint', path: '/oauth2-help', want: false },
		{ name: 'another API under a longer name', path: '/api/v1/accounts', want: false },
		{ name: 'the admin API', path: '/api/v1/admin/me', want: false }
	];

	for (const { name, path, want } of tests) {
		it(name, () => expect(isApiPath(path)).toBe(want));
	}
});
