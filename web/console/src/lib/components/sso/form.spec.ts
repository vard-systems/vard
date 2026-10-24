import { describe, expect, it } from 'vitest';
import type { SSOConnection } from '#lib/api/index.js';
import { SSOForm } from './form.svelte';

function oidc(overrides: Partial<SSOForm> = {}): SSOForm {
	return Object.assign(new SSOForm(), {
		name: 'Acme',
		issuer: 'https://acme.okta.com',
		clientId: 'client',
		domains: ['acme.test'],
		...overrides
	});
}

describe('SSOForm.missing', () => {
	it.each([
		['a new form', new SSOForm(), 'connection'],
		['an OIDC connection with no issuer', oidc({ issuer: ' ' }), 'connection'],
		['a SAML connection with no metadata', oidc({ protocol: 'saml' }), 'connection'],
		['no domain and no sign-in button', oidc({ domains: [], showOnLogin: false }), 'signin']
	])('%s names its tab', (_, form, tab) => expect(form.missing?.tab).toBe(tab));

	it('a complete OIDC connection lacks nothing', () => expect(oidc().missing).toBeNull());

	it('a SAML connection with metadata pasted lacks nothing', () =>
		expect(oidc({ protocol: 'saml', metadata: '<EntityDescriptor/>' }).missing).toBeNull());
});

describe('SSOForm.toInput', () => {
	it('gives the protocol and identifier only when creating', () => {
		const form = oidc({ slug: ' acme ' });

		expect(form.toInput(true)).toMatchObject({ protocol: 'oidc', slug: 'acme' });
		expect(form.toInput(false)).not.toHaveProperty('protocol');
		expect(form.toInput(false)).not.toHaveProperty('slug');
	});

	it('keeps the stored client secret when none is typed', () => {
		expect(oidc().toInput(false)).not.toHaveProperty('client_secret');
		expect(oidc({ clientSecret: ' new ' }).toInput(false)).toMatchObject({ client_secret: 'new' });
	});

	it('requires SSO only of an address at some domain', () => {
		expect(oidc({ enforceDomains: true }).toInput(false).enforce_domains).toBe(true);
		expect(
			oidc({ enforceDomains: true, domains: [], showOnLogin: true }).toInput(false).enforce_domains
		).toBe(false);
	});

	it('sends pasted SAML metadata only without a metadata URL', () => {
		const pasted = oidc({ protocol: 'saml', metadata: '<xml/>' }).toInput(false);
		const fetched = oidc({
			protocol: 'saml',
			metadata: '<xml/>',
			metadataUrl: 'https://idp.test/metadata'
		}).toInput(false);

		expect(pasted).toMatchObject({ metadata: '<xml/>', metadata_url: '' });
		expect(fetched).not.toHaveProperty('metadata');
		expect(fetched).not.toHaveProperty('issuer');
	});

	it('leaves out a role mapping with no group', () => {
		const form = oidc({
			mappings: [
				{ group: 'admins', role_id: 'r1' },
				{ group: ' ', role_id: 'r2' }
			]
		});

		expect(form.toInput(false).role_mappings).toEqual([{ group: 'admins', role_id: 'r1' }]);
	});

	it('splits scopes on any space', () =>
		expect(oidc({ scopes: ' email  profile\tgroups ' }).toInput(false).scopes).toEqual([
			'email',
			'profile',
			'groups'
		]));
});

describe('SSOForm.fill', () => {
	it('does not show metadata the server read from an address as typed', () => {
		const form = new SSOForm({
			protocol: 'saml',
			metadata_url: 'https://idp.test/metadata',
			metadata: '<xml/>'
		} as SSOConnection);

		expect(form.metadata).toBe('');
		expect(form.metadataUrl).toBe('https://idp.test/metadata');
	});
});
