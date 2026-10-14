import { describe, expect, it } from 'vitest';
import { translator } from '#lib/i18n/index.js';
import { base } from '#lib/i18n/messages.js';
import { describeDevice, formatDate, initials, timeAgo } from './format';

const english = translator(
	() => base,
	() => 'en'
);
const russian = translator(
	() => ({
		'device.on': '{browser} на {system}',
		'device.unknown_browser': 'Неизвестный браузер',
		'device.unknown_system': 'неизвестном устройстве'
	}),
	() => 'ru'
);

describe('describeDevice', () => {
	it('names common browsers and systems', () => {
		const chromeMac =
			'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0 Safari/537.36';
		const safariPhone =
			'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1';

		expect(describeDevice(chromeMac, english)).toBe('Chrome on macOS');
		expect(describeDevice(safariPhone, english)).toBe('Safari on iPhone');
		expect(describeDevice('', english)).toBe('Unknown browser on an unknown device');
	});

	it("says it in the page's language", () => {
		expect(describeDevice('', russian)).toBe('Неизвестный браузер на неизвестном устройстве');
	});
});

describe('timeAgo', () => {
	const now = Date.parse('2026-09-15T12:00:00Z');

	it('says how long ago', () => {
		expect(timeAgo('2026-09-15T11:59:30Z', english, now)).toBe('now');
		expect(timeAgo('2026-09-15T11:55:00Z', english, now)).toBe('5 minutes ago');
		expect(timeAgo('2026-09-12T12:00:00Z', english, now)).toBe('3 days ago');
	});

	it("says it in the page's language", () => {
		expect(timeAgo('2026-09-12T12:00:00Z', russian, now)).toBe('3 дня назад');
	});
});

describe('formatDate', () => {
	it("says it in the page's language", () => {
		expect(formatDate('2026-09-15T12:00:00Z', english, 'UTC')).toBe('Sep 15, 2026');
		expect(formatDate(null, english, 'UTC')).toBe('—');
	});

	it("names the day in the organisation's zone", () => {
		expect(formatDate('2026-09-15T20:00:00Z', english, 'UTC')).toBe('Sep 15, 2026');
		expect(formatDate('2026-09-15T20:00:00Z', english, 'Asia/Bishkek')).toBe('Sep 16, 2026');
	});
});

describe('initials', () => {
	it('uses the name, or the email', () => {
		expect(initials({ first_name: 'Ada', last_name: 'Lovelace', email: 'a@x' })).toBe('AL');
		expect(initials({ first_name: '', last_name: '', email: 'grace@x' })).toBe('G');
	});
});
