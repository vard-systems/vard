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

describe('toApi and the client certificate', () => {
	const send = async (certificateHeader: string, sent: Record<string, string>) => {
		vi.resetModules();
		vi.doMock('$app/env/private', () => ({
			API_URL: 'http://api.test',
			CLIENT_CERT_HEADER: certificateHeader
		}));
		const { toApi } = await import('./api');

		let forwarded = new Headers();
		vi.stubGlobal('fetch', async (_: URL, init: RequestInit) => {
			forwarded = new Headers(init.headers);
			return new Response(null, { status: 200 });
		});

		const url = new URL('http://id.test/oauth2/token');
		await toApi({
			request: new Request(url, { method: 'POST', headers: sent, body: '' }),
			url,
			getClientAddress: () => '203.0.113.7'
		} as never);

		vi.unstubAllGlobals();
		return forwarded.get('x-client-certificate');
	};

	it('drops a certificate a caller sent, with no proxy in front', async () => {
		expect(await send('', { 'x-client-certificate': 'forged' })).toBeNull();
	});

	it("passes on the proxy's certificate", async () => {
		expect(await send('x-ssl-client-cert', { 'x-ssl-client-cert': 'from-the-proxy' })).toBe(
			'from-the-proxy'
		);
	});

	it("drops a caller's copy under the API's name when the proxy names another", async () => {
		expect(await send('x-ssl-client-cert', { 'x-client-certificate': 'forged' })).toBeNull();
	});
});
