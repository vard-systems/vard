import { describe, expect, it } from 'vitest';
import { secondFactorHref, socialHref, ssoHref } from './links';

describe('provider start links', () => {
	it.each([
		['a provider alone', socialHref('google', {}), '/oauth2/social/google/start'],
		[
			'a sign-in request and where to go next',
			socialHref('google', { request: 'abc', next: '/security' }),
			'/oauth2/social/google/start?request=abc&next=%2Fsecurity'
		],
		[
			'an SSO connection with the address typed',
			ssoHref('acme', { request: 'abc', email: 'a@acme.test' }),
			'/oauth2/sso/acme/start?request=abc&login_hint=a%40acme.test'
		],
		[
			'a slug that is not a plain path segment',
			ssoHref('a/../b', {}),
			'/oauth2/sso/a%2F..%2Fb/start'
		],
		['nulls left out', ssoHref('acme', { request: null, next: null }), '/oauth2/sso/acme/start']
	])('%s', (_, href, expected) => expect(href).toBe(expected));
});

describe('the second-factor page', () => {
	it.each([
		['for an application', secondFactorHref('abc', '/'), '/second-factor?request=abc'],
		[
			'for the account pages',
			secondFactorHref(null, '/security'),
			'/second-factor?next=%2Fsecurity'
		],
		['with nothing to carry', secondFactorHref(null, null), '/second-factor']
	])('%s', (_, href, expected) => expect(href).toBe(expected));
});
