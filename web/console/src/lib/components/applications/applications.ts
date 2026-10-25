import type { IconName } from '#lib/components/ui/index.js';
import type {
	ApplicationInput,
	ApplicationType,
	AuthMethod,
	GrantType,
	Scope,
	TLSSubjectKind
} from '#lib/api/index.js';

export const types: Record<
	ApplicationType,
	{ label: string; description: string; icon: IconName; public: boolean }
> = {
	web: {
		label: 'Web application',
		description: 'Rendered on a server that keeps a client secret',
		icon: 'ri-computer-line',
		public: false
	},
	spa: {
		label: 'Single-page app',
		description: 'Runs in the browser; no secret, PKCE instead',
		icon: 'ri-apps-line',
		public: true
	},
	native: {
		label: 'Native app',
		description: 'Mobile or desktop; no secret, PKCE instead',
		icon: 'ri-smartphone-line',
		public: true
	},
	m2m: {
		label: 'Machine to machine',
		description: 'A service calling APIs as itself, with no user',
		icon: 'ri-server-line',
		public: false
	}
};

export const authMethods: { value: AuthMethod; label: string; description: string }[] = [
	{
		value: 'client_secret_basic',
		label: 'client_secret_basic',
		description: 'The secret in an HTTP Basic header (recommended)'
	},
	{
		value: 'client_secret_post',
		label: 'client_secret_post',
		description: 'The secret in the token request body'
	},
	{
		value: 'tls_client_auth',
		label: 'tls_client_auth',
		description: 'A certificate from a trusted authority, over mutual TLS'
	},
	{
		value: 'self_signed_tls_client_auth',
		label: 'self_signed_tls_client_auth',
		description: 'A self-signed certificate registered here, over mutual TLS'
	}
];

/** Which certificate field names a tls_client_auth client; exactly one is registered. */
export const tlsSubjectKinds: {
	value: Exclude<TLSSubjectKind, ''>;
	label: string;
	description: string;
}[] = [
	{
		value: 'subject_dn',
		label: 'Subject DN',
		description:
			'Exactly as RFC 4514 writes it, most specific first: CN=app,O=Example. Compared as text, so a SAN is more reliable.'
	},
	{ value: 'san_dns', label: 'DNS name', description: 'A dNSName subject alternative name' },
	{ value: 'san_uri', label: 'URI', description: 'A uniformResourceIdentifier alternative name' },
	{ value: 'san_ip', label: 'IP address', description: 'An iPAddress alternative name' },
	{ value: 'san_email', label: 'Email address', description: 'An rfc822Name alternative name' }
];

/** The certificates in pasted text, one PEM block each. */
export function certificates(text: string): string[] {
	return text.match(/-----BEGIN CERTIFICATE-----[\s\S]*?-----END CERTIFICATE-----/g) ?? [];
}

export const grants: { value: GrantType; description: string }[] = [
	{ value: 'authorization_code', description: 'Sign users in with a redirect and a code' },
	{ value: 'refresh_token', description: 'Get new tokens without asking the user again' },
	{ value: 'client_credentials', description: 'Act as the application itself, with no user' },
	{
		value: 'urn:ietf:params:oauth:grant-type:device_code',
		description: 'Sign users in on a TV or a command line: they enter a short code on their phone'
	}
];

/** The grants that get tokens for a user. */
export const userGrants: GrantType[] = [
	'authorization_code',
	'urn:ietf:params:oauth:grant-type:device_code'
];

export const scopes: { value: Scope; description: string }[] = [
	{ value: 'openid', description: 'Sign in with OpenID Connect and get an ID token' },
	{ value: 'profile', description: "The user's name" },
	{ value: 'email', description: "The user's address and whether it is verified" },
	{ value: 'offline_access', description: 'A refresh token, to stay signed in' },
	{ value: 'roles', description: "The user's roles in this application" }
];

/** The settings a new application of a type starts with: the choices RFC
    9700 recommends, so the form is right before anything is changed. */
export function blank(type: ApplicationType): ApplicationInput {
	return {
		name: '',
		description: '',
		type,
		logo_url: '',
		website_url: '',
		privacy_url: '',
		terms_url: '',
		token_auth_method: types[type].public ? 'none' : 'client_secret_basic',
		grant_types: type === 'm2m' ? ['client_credentials'] : ['authorization_code', 'refresh_token'],
		redirect_uris: [],
		post_logout_redirect_uris: [],
		allow_development_redirects: false,
		backchannel_logout_uri: '',
		frontchannel_logout_uri: '',
		scopes: type === 'm2m' ? [] : ['openid', 'profile', 'email'],
		require_pkce: type !== 'm2m',
		require_pushed_authorization_requests: false,
		tls_subject_kind: '',
		tls_subject: '',
		tls_certificates: [],
		tls_client_certificate_bound_access_tokens: false,
		access_token_lifetime: 60 * 60,
		id_token_lifetime: 60 * 60,
		refresh_token_lifetime: 30 * 24 * 60 * 60,
		assert_roles: true,
		require_role_assignment: false,
		is_enabled: true,
		allow_registration: type !== 'm2m',
		login_flow_id: ''
	};
}

export function lines(text: string): string[] {
	return text
		.split('\n')
		.map((line) => line.trim())
		.filter((line) => line !== '');
}

export const ADMIN_CLI = 'admin-cli';
