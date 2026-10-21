import { request, type Fetch } from './client';

export type Application = {
	name: string;
	logo_url: string;
	website_url: string;
	privacy_url: string;
	terms_url: string;
	allow_registration: boolean;
};

export type Organization = {
	name: string;
	logo_url: string;
	support_email: string;
	support_phone: string;
	terms_url: string;
	privacy_url: string;
	timezone: string;
};

export type LoginStep =
	'identifier' | 'password' | 'social' | 'email_code' | 'totp' | 'terms' | 'consent';

/**
 * What these pages may offer, from the application's login flow or the default. The server
 * enforces it regardless.
 */
export type LoginOptions = {
	steps: LoginStep[];
	allow_sign_in: boolean;
	allow_registration: boolean;
	allow_password_reset: boolean;
	allow_remember_me: boolean;
	allow_email_change: boolean;
};

export type PublicLanguage = {
	code: string;
	name: string;
	native_name: string;
};

export type SocialProvider = {
	slug: string;
	name: string;
	kind: string;
};

export type SSOConnection = {
	slug: string;
	name: string;
	protocol: 'oidc' | 'saml';
};

export type SignInRequest = {
	application: Application;
	login_hint: string;
	expires_at: string;
};

export type CodeChallenge = {
	/** Names the sign-in. The page keeps it and sends it back with the code;
	    the code alone is no use without it. */
	handle: string;
	email: string;
	expires_at: string;
	resend_after: number;
	attempts_left: number;
};

export type SignedIn = {
	redirect_to?: string;
	password_change_required?: boolean;
	reset_token?: string;
	code?: CodeChallenge;
};

export type User = {
	id: string;
	email: string;
	email_verified: boolean;
	first_name: string;
	last_name: string;
	created_at: string;
	last_login_at: string | null;
};

export type Session = {
	id: string;
	ip: string;
	user_agent: string;
	signed_in_at: string;
	expires_at: string;
	current: boolean;
};

export type ConnectedApplication = {
	client_id: string;
	name: string;
	logo_url: string;
	website_url: string;
	scopes: string[];
	authorized_at: string;
	last_used_at: string;
};

export type RegisterInput = {
	request: string;
	email: string;
	password: string;
	first_name: string;
	last_name: string;
	accept_terms: boolean;
	remember: boolean;
};

const encode = encodeURIComponent;

export const signIn = {
	request: (handle: string, fetch?: Fetch) =>
		request<SignInRequest>(`/account/requests/${encode(handle)}`, { fetch }),

	application: (clientId: string, fetch?: Fetch) =>
		request<{ application: Application }>(`/account/applications/${encode(clientId)}`, { fetch }),

	organization: (fetch?: Fetch) =>
		request<{ organization: Organization }>('/account/organization', { fetch }),

	socialProviders: (fetch?: Fetch) =>
		request<{ providers: SocialProvider[] }>('/account/social-providers', { fetch }),

	ssoConnections: (fetch?: Fetch) =>
		request<{ connections: SSOConnection[]; available: boolean }>('/account/sso', { fetch }),

	discoverSSO: (email: string) =>
		request<{ connection: SSOConnection; enforced: boolean }>('/account/sso/discover', {
			method: 'POST',
			body: { email }
		}),

	languages: (fetch?: Fetch) =>
		request<{ languages: PublicLanguage[]; default: string }>('/account/languages', { fetch }),

	languageText: (code: string, fetch?: Fetch) =>
		request<{ language: PublicLanguage; messages: Record<string, string> }>(
			`/account/languages/${encodeURIComponent(code)}`,
			{ fetch }
		),

	loginOptions: (handle: string, fetch?: Fetch) =>
		request<{ login: LoginOptions }>(
			handle ? `/account/login-options?request=${encode(handle)}` : '/account/login-options',
			{ fetch }
		),

	login: (body: { request: string; email: string; password: string; remember: boolean }) =>
		request<SignedIn>('/account/login', { method: 'POST', body }),

	register: (body: RegisterInput) =>
		request<SignedIn>('/account/register', { method: 'POST', body }),

	code: (body: { handle: string; code: string }) =>
		request<SignedIn>('/account/login/code', { method: 'POST', body }),

	resendCode: (body: { handle: string }) =>
		request<{ code: CodeChallenge }>('/account/login/code/resend', { method: 'POST', body }),

	forgotPassword: (body: { request: string; email: string; language: string }) =>
		request<{ status: string }>('/account/forgot-password', { method: 'POST', body }),

	checkReset: (token: string, fetch?: Fetch) =>
		request<{ valid: boolean }>(`/account/reset-password?token=${encode(token)}`, { fetch }),

	resetPassword: (body: { token: string; password: string }) =>
		request<{ status: string }>('/account/reset-password', { method: 'POST', body }),

	verifyEmail: (body: { token: string }) =>
		request<{ status: string }>('/account/verify-email', { method: 'POST', body }),

	/** Signs this browser out. The confirmation page passes on what the
	    application's logout named; `location` is where the server says to go. */
	logout: (body?: { client_id: string; post_logout_redirect_uri: string; state: string }) =>
		request<{ status: string; location: string }>('/account/logout', { method: 'POST', body })
};

export const account = {
	me: (fetch?: Fetch) => request<{ user: User }>('/account/me', { fetch }),

	updateProfile: (body: { first_name: string; last_name: string }) =>
		request<{ user: User }>('/account/me', { method: 'PATCH', body }),

	changePassword: (body: { current_password: string; new_password: string }) =>
		request<{ status: string }>('/account/password', { method: 'POST', body }),

	changeEmail: (body: { email: string; current_password: string }) =>
		request<{ status: string }>('/account/email', { method: 'POST', body }),

	sessions: (fetch?: Fetch) => request<{ sessions: Session[] }>('/account/sessions', { fetch }),

	endSession: (id: string) =>
		request<void>(`/account/sessions/${encode(id)}`, { method: 'DELETE' }),

	applications: (fetch?: Fetch) =>
		request<{ applications: ConnectedApplication[] }>('/account/connected-applications', {
			fetch
		}),

	disconnect: (clientId: string) =>
		request<void>(`/account/connected-applications/${encode(clientId)}`, { method: 'DELETE' })
};
