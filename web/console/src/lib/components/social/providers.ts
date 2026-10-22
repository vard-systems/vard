import type { IconName } from '#lib/components/ui/index.js';

import type { SocialKind, SocialProvider, SocialSpec } from '#lib/api/index.js';
import type { SelectOption } from '#lib/components/ui/index.js';

const marks: Record<SocialKind, IconName> = {
	google: 'ri-google-fill',
	apple: 'ri-apple-fill',
	facebook: 'ri-facebook-circle-fill',
	yandex: 'ri-key-2-line',
	vk: 'ri-vk-fill',
	oidc: 'ri-shield-keyhole-line',
	oauth2: 'ri-key-2-line'
};

export function markFor(kind: SocialKind): IconName {
	return marks[kind] ?? 'ri-key-2-line';
}

export function kindOptions(kinds: SocialSpec[]): SelectOption<SocialKind>[] {
	return kinds.map((kind) => ({
		value: kind.kind,
		label: kind.label,
		icon: markFor(kind.kind),
		description: kind.custom
			? 'Any provider, with its own addresses'
			: `Signed in at ${hostOf(kind.authorize_url)}`
	}));
}

export const tokenAuthOptions: SelectOption<'' | 'basic' | 'post'>[] = [
	{
		value: '',
		label: "The provider's usual way",
		description: 'What this kind of provider expects'
	},
	{ value: 'basic', label: 'HTTP Basic', description: 'Every OAuth 2.0 server accepts this' },
	{ value: 'post', label: 'In the request body', description: 'client_secret as a form field' }
];

export function hostOf(address: string): string {
	try {
		return new URL(address).host;
	} catch {
		return address;
	}
}

export function missingFrom(provider: SocialProvider, spec: SocialSpec | undefined): string {
	if (spec?.signed_secret) {
		if (!provider.has_private_key) return 'no signing key';
		if (!provider.team_id || !provider.key_id) return 'no team or key id';
		return '';
	}

	return provider.has_client_secret ? '' : 'no client secret';
}
