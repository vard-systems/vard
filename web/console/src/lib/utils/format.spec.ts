import { describe, expect, it } from 'vitest';

import {
	countOf,
	describeUserAgent,
	formatDateTime,
	formatDayHeading,
	formatRelative
} from './format';

describe('formatDateTime', () => {
	it('reads a timestamp the way the panel shows it', () => {
		const shown = formatDateTime('2026-09-13T14:38:09.654650+06:00');

		expect(shown).toContain('2026');
		expect(shown.length).toBeGreaterThan(0);
	});

	it('leaves something that is not a date alone', () => {
		expect(formatDateTime('not a date')).toBe('not a date');
	});

	it('has something to show for nothing', () => {
		expect(formatDateTime('')).toBe('');
	});
});

describe('formatRelative', () => {
	const now = new Date(2026, 8, 14, 15, 0, 0);
	const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();

	it('says recent moments in minutes and hours', () => {
		expect(formatRelative(ago(20_000), now)).toBe('just now');
		expect(formatRelative(ago(5 * 60_000), now)).toBe('5 min ago');
		expect(formatRelative(ago(3 * 3_600_000), now)).toBe('3 h ago');
	});

	it('counts calendar days after that', () => {
		expect(formatRelative(new Date(2026, 8, 13, 23, 0).toISOString(), now)).toBe('yesterday');
		expect(formatRelative(new Date(2026, 8, 10, 12, 0).toISOString(), now)).toBe('4 days ago');
	});
});

describe('formatDayHeading', () => {
	const now = new Date(2026, 8, 14, 9, 0, 0);

	it('names today and yesterday', () => {
		expect(formatDayHeading(new Date(2026, 8, 14, 1, 0).toISOString(), now)).toBe('Today');
		expect(formatDayHeading(new Date(2026, 8, 13, 22, 0).toISOString(), now)).toBe('Yesterday');
	});
});

describe('describeUserAgent', () => {
	it('names the browser and the system', () => {
		const chrome =
			'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0 Safari/537.36';
		const firefox =
			'Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:130.0) Gecko/20100101 Firefox/130.0';

		expect(describeUserAgent(chrome)).toBe('Chrome on macOS');
		expect(describeUserAgent(firefox)).toBe('Firefox on Windows');
		expect(describeUserAgent('')).toBe('Unknown device');
	});
});

describe('countOf', () => {
	it.each([
		[1, 'role', '1 role'],
		[0, 'role', '0 roles'],
		[3, 'API', '3 APIs'],
		[1200, 'event', '1,200 events']
	])('counts %i %s as "%s"', (n, noun, expected) => {
		expect(countOf(n, noun)).toBe(expected);
	});
});
