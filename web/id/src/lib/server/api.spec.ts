import type { RequestEvent } from '@sveltejs/kit';
import { describe, expect, it, vi } from 'vitest';

vi.mock('$app/env/private', () => ({ API_URL: 'http://api.test' }));

const { isApiPath, toApi } = await import('./api');

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

describe('toApi', () => {
	const event = (): RequestEvent =>
		({
			request: new Request('http://app.test/api/v1/x', { method: 'GET' }),
			url: new URL('http://app.test/api/v1/x'),
			getClientAddress: () => '192.0.2.1'
		}) as unknown as RequestEvent;

	const tests: { name: string; failure: unknown; want: number }[] = [
		{
			name: 'an API that does not answer in time',
			failure: new DOMException('timed out', 'TimeoutError'),
			want: 504
		},
		{ name: 'an API that cannot be reached', failure: new TypeError('fetch failed'), want: 502 }
	];

	for (const { name, failure, want } of tests) {
		it(name, async () => {
			vi.stubGlobal('fetch', vi.fn().mockRejectedValue(failure));
			expect((await toApi(event())).status).toBe(want);
			vi.unstubAllGlobals();
		});
	}

	it('gives up when the browser does, or when the API would have', async () => {
		const fetch = vi.fn().mockResolvedValue(new Response('ok'));
		vi.stubGlobal('fetch', fetch);
		await toApi(event());
		vi.unstubAllGlobals();

		expect(fetch.mock.calls[0][1].signal).toBeInstanceOf(AbortSignal);
	});
});
