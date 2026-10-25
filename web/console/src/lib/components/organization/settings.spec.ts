import { describe, expect, it } from 'vitest';
import type { Organization } from '#lib/api/index.js';
import {
	changed,
	logoProblem,
	offsetOf,
	publicDetails,
	settingsOf,
	tidy,
	timezoneOptions
} from './settings';

const saved: Organization = {
	name: 'Northwind',
	slug: 'northwind',
	logo_version: '',
	support_email: 'help@northwind.test',
	support_phone: '',
	terms_url: '',
	privacy_url: '',
	timezone: 'Asia/Bishkek',
	created_at: '2026-01-01T00:00:00Z'
};

describe('the organization form', () => {
	it('edits everything but when the record was made and the logo, saved on its own', () => {
		const settings = settingsOf({ ...saved, logo_version: 'a1' });
		expect(settings).not.toHaveProperty('created_at');
		expect(settings).not.toHaveProperty('logo_version');
	});

	it('sends trimmed values and a lower-case identifier', () =>
		expect(tidy({ ...settingsOf(saved), name: ' Northwind ', slug: ' NorthWind ' })).toMatchObject({
			name: 'Northwind',
			slug: 'northwind'
		}));

	it('is unchanged until a value differs from what is saved', () => {
		expect(changed(tidy(settingsOf(saved)), saved)).toBe(false);
		expect(
			changed({ ...settingsOf(saved), terms_url: 'https://northwind.test/terms' }, saved)
		).toBe(true);
	});

	it('counts the public details filled in', () => {
		const filled = (organization: Organization) =>
			publicDetails(settingsOf(organization), organization)
				.filter((it) => it.filled)
				.map((it) => it.label);

		expect(filled(saved)).toEqual(['Support email']);
		expect(filled({ ...saved, logo_version: 'a1' })).toEqual(['Logo', 'Support email']);
	});

	it.each([
		['a PNG', { type: 'image/png', size: 2048 }, ''],
		['a JPEG at the limit', { type: 'image/jpeg', size: 512 * 1024 }, ''],
		['an SVG', { type: 'image/svg+xml', size: 2048 }, 'Choose a PNG or JPEG file.'],
		['a GIF', { type: 'image/gif', size: 2048 }, 'Choose a PNG or JPEG file.'],
		[
			'a PNG over 512 KB',
			{ type: 'image/png', size: 512 * 1024 + 1 },
			'The file is larger than 512 KB. Choose a smaller one.'
		]
	])('takes or refuses %s before sending it', (_, file, expected) =>
		expect(logoProblem(file)).toBe(expected)
	);

	it('offers UTC and the saved zone first, once each', () => {
		const zones = timezoneOptions('Asia/Bishkek').map((it) => it.value);
		expect(zones.slice(0, 2)).toEqual(['UTC', 'Asia/Bishkek']);
		expect(zones.filter((zone) => zone === 'Asia/Bishkek')).toHaveLength(1);
	});

	it.each([
		['a zone with an offset', 'Asia/Bishkek', 'GMT+06:00'],
		['a zone nobody knows', 'Nowhere/Atlantis', '']
	])('writes the offset of %s', (_, zone, expected) =>
		expect(offsetOf(zone, new Date('2026-06-01T00:00:00Z'))).toBe(expected)
	);
});
