import { isHttpError } from '@sveltejs/kit';
import { describe, expect, it } from 'vitest';
import { currentUser } from './session';

function answering(status: number, body: unknown = {}): typeof fetch {
	return async () => new Response(JSON.stringify(body), { status });
}

describe('currentUser', () => {
	it('is nobody when the API says there is no session', async () => {
		expect(await currentUser(answering(401))).toBeNull();
	});

	it.each([
		['the API failing', answering(500), 500],
		['the API out of reach behind this server', answering(502), 502],
		[
			'the network gone',
			(async () => {
				throw new TypeError('offline');
			}) as typeof fetch,
			503
		]
	])('answers %s with an error page, not a crash', async (_, fetcher, status) => {
		const failure = await currentUser(fetcher).catch((err: unknown) => err);
		expect(isHttpError(failure) && failure.status).toBe(status);
	});
});
