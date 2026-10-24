import { describe, expect, it } from 'vitest';
import { blank } from './applications';
import { ApplicationForm } from './form.svelte';

function filled(type: Parameters<typeof blank>[0] = 'web'): ApplicationForm {
	const form = new ApplicationForm();
	form.fill({ ...blank(type), name: ' Shop ', description: ' A shop ' });
	return form;
}

describe('ApplicationForm', () => {
	it('edits lifetimes in minutes and days and sends seconds', () => {
		const form = filled();
		expect([form.accessMinutes, form.idMinutes, form.refreshDays]).toEqual(['60', '60', '30']);

		form.accessMinutes = '15';
		form.refreshDays = '7';

		expect(form.toInput()).toMatchObject({
			access_token_lifetime: 15 * 60,
			id_token_lifetime: 60 * 60,
			refresh_token_lifetime: 7 * 24 * 60 * 60
		});
	});

	it('sends one redirect URI a line, blank lines left out', () => {
		const form = filled();
		form.redirects = ' https://shop.test/a \n\n https://shop.test/b';

		expect(form.toInput().redirect_uris).toEqual(['https://shop.test/a', 'https://shop.test/b']);
	});

	it('trims the name and description', () =>
		expect(filled().toInput()).toMatchObject({ name: 'Shop', description: 'A shop' }));

	it('keeps how it is presented when the type changes, and nothing else', () => {
		const form = filled('web');
		form.input.logo_url = 'https://shop.test/logo.png';
		form.redirects = 'https://shop.test/callback';

		form.changeType('m2m');

		expect(form.machine).toBe(true);
		expect(form.input).toMatchObject({
			name: ' Shop ',
			logo_url: 'https://shop.test/logo.png',
			grant_types: ['client_credentials']
		});
		expect(form.redirects).toBe('');
	});

	it('adds a grant once and takes it away', () => {
		const form = filled();
		form.setGrant('refresh_token', true);
		expect(form.input.grant_types).toEqual(['authorization_code', 'refresh_token']);

		form.setGrant('refresh_token', false);
		expect(form.input.grant_types).toEqual(['authorization_code']);
	});

	it('is ready once it has a name', () => {
		const form = new ApplicationForm();
		expect(form.ready).toBe(false);

		form.input.name = 'Shop';
		expect(form.ready).toBe(true);
	});
});
