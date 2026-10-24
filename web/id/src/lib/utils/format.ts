import type { Translate } from '#lib/i18n/index.js';

export function formatDate(
	value: string | null | undefined,
	t: Translate,
	timeZone: string
): string {
	if (!value) return '—';
	return new Date(value).toLocaleDateString(t.language, {
		day: 'numeric',
		month: 'short',
		year: 'numeric',
		timeZone
	});
}

const units: [number, Intl.RelativeTimeFormatUnit][] = [
	[60 * 60 * 24 * 365, 'year'],
	[60 * 60 * 24 * 30, 'month'],
	[60 * 60 * 24, 'day'],
	[60 * 60, 'hour'],
	[60, 'minute']
];

export function timeAgo(value: string, t: Translate, now = Date.now()): string {
	const seconds = Math.max(0, Math.round((now - new Date(value).getTime()) / 1000));
	const format = new Intl.RelativeTimeFormat(t.language, { numeric: 'auto' });

	for (const [size, unit] of units) {
		const count = Math.floor(seconds / size);
		if (count >= 1) return format.format(-count, unit);
	}

	return format.format(0, 'second');
}

export function describeDevice(userAgent: string, t: Translate): string {
	const ua = userAgent || '';

	const browser =
		[
			['Edge', /Edg\//],
			['Opera', /OPR\//],
			['Firefox', /Firefox\//],
			['Chrome', /Chrome\//],
			['Safari', /Safari\//]
		].find(([, pattern]) => (pattern as RegExp).test(ua))?.[0] ?? t('device.unknown_browser');

	const system =
		[
			['iPhone', /iPhone/],
			['iPad', /iPad/],
			['Android', /Android/],
			['Windows', /Windows/],
			['macOS', /Mac OS X|Macintosh/],
			['Linux', /Linux/]
		].find(([, pattern]) => (pattern as RegExp).test(ua))?.[0] ?? t('device.unknown_system');

	return t('device.on', { browser: String(browser), system: String(system) });
}

type Person = { first_name: string; last_name: string; email: string };

/** A person's name, or their address when they have not given one. */
export function displayName(user: Person): string {
	return `${user.first_name} ${user.last_name}`.trim() || user.email;
}

export function initials(user: Person): string {
	const letters = `${user.first_name.charAt(0)}${user.last_name.charAt(0)}`.trim();
	return (letters || user.email.charAt(0)).toUpperCase();
}
