import type { Organization, OrganizationSettings } from '#lib/api/index.js';

/** The part of the organization the form edits. */
export function settingsOf({ created_at, ...settings }: Organization): OrganizationSettings {
	void created_at;
	return settings;
}

/** What the form sends: trimmed, the identifier in lower case. */
export function tidy(form: OrganizationSettings): OrganizationSettings {
	return {
		name: form.name.trim(),
		slug: form.slug.trim().toLowerCase(),
		logo_url: form.logo_url.trim(),
		support_email: form.support_email.trim(),
		support_phone: form.support_phone.trim(),
		terms_url: form.terms_url.trim(),
		privacy_url: form.privacy_url.trim(),
		timezone: form.timezone
	};
}

/** Whether anything sent differs from what is saved. */
export function changed(input: OrganizationSettings, saved: Organization): boolean {
	return Object.entries(input).some(
		([key, value]) => value !== saved[key as keyof OrganizationSettings]
	);
}

/** The details the sign-in pages show only once they are filled in. */
export function publicDetails(input: OrganizationSettings): { label: string; filled: boolean }[] {
	return [
		{ label: 'Logo', filled: input.logo_url !== '' },
		{ label: 'Support email', filled: input.support_email !== '' },
		{ label: 'Support number', filled: input.support_phone !== '' },
		{ label: 'Terms', filled: input.terms_url !== '' },
		{ label: 'Privacy policy', filled: input.privacy_url !== '' }
	];
}

/** Every timezone this browser knows, UTC and the saved one first, each once. */
export function timezoneOptions(saved: string): { value: string; label: string }[] {
	const zones = ['UTC', saved, ...Intl.supportedValuesOf('timeZone')];
	return zones
		.filter((zone, index) => zone !== '' && zones.indexOf(zone) === index)
		.map((zone) => ({ value: zone, label: zone.replaceAll('_', ' ') }));
}

/** A zone's offset from UTC now, as "GMT+06:00"; "" for a zone this browser does not know. */
export function offsetOf(timezone: string, now = new Date()): string {
	try {
		return (
			new Intl.DateTimeFormat('en-US', { timeZone: timezone, timeZoneName: 'longOffset' })
				.formatToParts(now)
				.find((part) => part.type === 'timeZoneName')?.value ?? ''
		);
	} catch {
		return '';
	}
}
