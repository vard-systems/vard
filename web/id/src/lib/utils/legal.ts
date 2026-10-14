import type { Application, Organization } from '#lib/api/index.js';

export type LegalLink = { key: string; href: string };

export function legalLinks(
	application: Application | null | undefined,
	organization: Organization | null | undefined
): LegalLink[] {
	const terms = application?.terms_url || organization?.terms_url || '';
	const privacy = application?.privacy_url || organization?.privacy_url || '';

	return [
		terms ? { key: 'legal.terms', href: terms } : null,
		privacy ? { key: 'legal.privacy', href: privacy } : null
	].filter((link) => link !== null);
}

export type SupportLink = { label: string; href: string };

export function supportLinks(organization: Organization | null | undefined): SupportLink[] {
	return [
		organization?.support_email
			? { label: organization.support_email, href: `mailto:${organization.support_email}` }
			: null,
		organization?.support_phone
			? {
					label: organization.support_phone,
					href: `tel:${organization.support_phone.replace(/[^+0-9]/g, '')}`
				}
			: null
	].filter((link) => link !== null);
}
