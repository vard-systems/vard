import { describe, expect, it } from 'vitest';

import type { Application, Organization } from '#lib/api/index.js';
import { legalLinks, supportLinks } from './legal';

const application = (over: Partial<Application> = {}): Application => ({
	name: 'Shop',
	logo_url: '',
	website_url: '',
	privacy_url: '',
	terms_url: '',
	allow_registration: true,
	...over
});

const organization = (over: Partial<Organization> = {}): Organization => ({
	name: 'Acme',
	logo_url: '',
	support_email: '',
	support_phone: '',
	terms_url: '',
	privacy_url: '',
	timezone: 'UTC',
	...over
});

describe('legalLinks', () => {
	const org = organization({
		terms_url: 'https://acme.example.com/terms',
		privacy_url: 'https://acme.example.com/privacy'
	});

	it("prefers the application's own agreements", () => {
		const links = legalLinks(
			application({
				terms_url: 'https://shop.example.com/terms',
				privacy_url: 'https://shop.example.com/privacy'
			}),
			org
		);

		expect(links.map((link) => link.href)).toEqual([
			'https://shop.example.com/terms',
			'https://shop.example.com/privacy'
		]);
	});

	it("falls back to the organisation's, one link at a time", () => {
		const links = legalLinks(application({ terms_url: 'https://shop.example.com/terms' }), org);

		expect(links.map((link) => link.href)).toEqual([
			'https://shop.example.com/terms',
			'https://acme.example.com/privacy'
		]);
	});

	it('shows the organisation alone when there is no application', () => {
		expect(legalLinks(null, org)).toHaveLength(2);
	});

	it('shows nothing nobody has published', () => {
		expect(legalLinks(application(), organization())).toEqual([]);
	});
});

describe('supportLinks', () => {
	it('writes an address and dials a number', () => {
		const links = supportLinks(
			organization({ support_email: 'help@acme.example.com', support_phone: '+996 555 123456' })
		);

		expect(links.map((link) => link.href)).toEqual([
			'mailto:help@acme.example.com',
			'tel:+996555123456'
		]);
	});

	it('has nothing to say for an organisation with no contact', () => {
		expect(supportLinks(organization())).toEqual([]);
		expect(supportLinks(null)).toEqual([]);
	});
});
