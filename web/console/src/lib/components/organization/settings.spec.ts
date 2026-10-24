import { describe, expect, it } from 'vitest';
import type { Organization } from '#lib/api/index.js';
import { changed, offsetOf, publicDetails, settingsOf, tidy, timezoneOptions } from './settings';

const saved: Organization = {
	name: 'Northwind',
	slug: 'northwind',
	logo_url: '',
	support_email: 'help@northwind.test',
	support_phone: '',
	terms_url: '',
	privacy_url: '',
	timezone: 'Asia/Bishkek',
	created_at: '2026-01-01T00:00:00Z'
};

describe('the organization form', () => {
	it('edits everything but when the record was made', () =>
		expect(settingsOf(saved)).not.toHaveProperty('created_at'));

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

	it('counts the public details filled in', () =>
		expect(
			publicDetails(settingsOf(saved))
				.filter((it) => it.filled)
				.map((it) => it.label)
		).toEqual(['Support email']));

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
