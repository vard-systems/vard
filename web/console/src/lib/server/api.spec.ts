import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/env/private', () => ({ API_URL: 'http://api.test' }));

const { isApiPath } = await import('./api');

describe('isApiPath', () => {
	const tests: { name: string; path: string; want: boolean }[] = [
		{ name: 'an admin endpoint', path: '/api/v1/admin/users', want: true },
		{ name: 'a page', path: '/users', want: false },
		{ name: 'another API under a longer name', path: '/api/v1/administrators', want: false },
		{ name: 'the account API', path: '/api/v1/account/me', want: false }
	];

	for (const { name, path, want } of tests) {
		it(name, () => expect(isApiPath(path)).toBe(want));
	}
});
