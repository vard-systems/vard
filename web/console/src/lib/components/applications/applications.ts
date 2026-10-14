import type { ComponentType } from 'svelte';
import { RiAppsLine, RiComputerLine, RiServerLine, RiSmartphoneLine } from 'svelte-remixicon';
import type {
	ApplicationInput,
	ApplicationType,
	AuthMethod,
	GrantType,
	Scope
} from '#lib/api/index.js';

export const types: Record<
	ApplicationType,
	{ label: string; description: string; icon: ComponentType; public: boolean }
> = {
	web: {
		label: 'Web application',
		description: 'Rendered on a server that keeps a client secret',
		icon: RiComputerLine,
		public: false
	},
	spa: {
		label: 'Single-page app',
		description: 'Runs in the browser; no secret, PKCE instead',
		icon: RiAppsLine,
		public: true
	},
	native: {
		label: 'Native app',
		description: 'Mobile or desktop; no secret, PKCE instead',
		icon: RiSmartphoneLine,
		public: true
	},
	m2m: {
		label: 'Machine to machine',
		description: 'A service calling APIs as itself, with no user',
		icon: RiServerLine,
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
	}
];

export const grants: { value: GrantType; description: string }[] = [
	{ value: 'authorization_code', description: 'Sign users in with a redirect and a code' },
	{ value: 'refresh_token', description: 'Get new tokens without asking the user again' },
	{ value: 'client_credentials', description: 'Act as the application itself, with no user' }
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
		scopes: type === 'm2m' ? [] : ['openid', 'profile', 'email'],
		require_pkce: type !== 'm2m',
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
