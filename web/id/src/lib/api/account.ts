import { request, type Fetch } from './client';

/** What an application's sign-in pages show about it. */
export type Application = {
	name: string;
	logo_url: string;
	website_url: string;
	/** The privacy policy, linked under the card and agreed to on registration. */
	privacy_url: string;
	/** The terms of service, the same. */
	terms_url: string;
	allow_registration: boolean;
};

/**
 * The organisation the server signs users in for, shown when an application has no name, logo
 * or links of its own. Empty values are omitted.
 */
export type Organization = {
	name: string;
	logo_url: string;
	support_email: string;
	support_phone: string;
	terms_url: string;
	privacy_url: string;
	/** The IANA zone dates are said in, "Asia/Bishkek". */
	timezone: string;
};

/** One step a login flow is made of. The names mirror the constants in
    internal/model/login_flow.go. */
export type LoginStep =
	'identifier' | 'password' | 'social' | 'email_code' | 'totp' | 'terms' | 'consent';

/**
 * What these pages may offer, from the application's login flow or the default. The server
 * enforces it regardless.
 */
export type LoginOptions = {
	steps: LoginStep[];
	/** False is a closed door: the pages say so rather than asking for
	    anything, and the server refuses every way in. */
	allow_sign_in: boolean;
	/** Offer "Create an account". */
	allow_registration: boolean;
	/** Offer "Forgotten your password". */
	allow_password_reset: boolean;
	/** Offer "Stay signed in" beside the password. Without it the session
	    ends when the browser closes. */
	allow_remember_me: boolean;
	/** Offer "Change" beside the address on the account page. */
	allow_email_change: boolean;
};

/** One language the sign-in pages may be shown in, as the picker lists it. */
export type PublicLanguage = {
	code: string;
	/** The language in English, and in itself. */
	name: string;
	native_name: string;
};

/** An account elsewhere that can be signed in with, as its button. */
export type SocialProvider = {
	slug: string;
	name: string;
	kind: string;
};

/** An organisation's identity provider, as the sign-in page offers it. */
export type SSOConnection = {
	slug: string;
	name: string;
	protocol: 'oidc' | 'saml';
};

/** A sign-in under way, known to the pages by the handle in `?request=`. */
export type SignInRequest = {
	application: Application;
	login_hint: string;
	expires_at: string;
};

/** A sign-in held back for a code emailed to the address on the account:
    what the page comes back with, where the message went, and what it may do
    while it waits. */
export type CodeChallenge = {
	/** Names the sign-in. The page keeps it and sends it back with the code;
	    the code alone is no use without it. */
	handle: string;
	/** The address the message went to, with the middle of it hidden. */
	email: string;
	expires_at: string;
	/** Seconds until another message may be asked for. */
	resend_after: number;
	/** Codes that may still be typed before the sign-in is over. */
	attempts_left: number;
};

/** What signing in or registering says to do next. */
export type SignedIn = {
	/** Back to the application, with a code. */
	redirect_to?: string;
	/** The password was a temporary one; choose a new one with this token. */
	password_change_required?: boolean;
	reset_token?: string;
	/** The login flow has the emailed code step: ask for the code in this,
	    and send it to `signIn.code`. Nobody is signed in until then. */
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

/** Signing in, registering and resetting a password: no session needed. */
export const signIn = {
	request: (handle: string, fetch?: Fetch) =>
		request<SignInRequest>(`/account/requests/${encode(handle)}`, { fetch }),

	application: (clientId: string, fetch?: Fetch) =>
		request<{ application: Application }>(`/account/applications/${encode(clientId)}`, { fetch }),

	organization: (fetch?: Fetch) =>
		request<{ organization: Organization }>('/account/organization', { fetch }),

	socialProviders: (fetch?: Fetch) =>
		request<{ providers: SocialProvider[] }>('/account/social-providers', { fetch }),

	/** The organisations' identity providers the sign-in page shows a button
	    for. */
	ssoConnections: (fetch?: Fetch) =>
		request<{ connections: SSOConnection[]; available: boolean }>('/account/sso', { fetch }),

	/** The identity provider an address signs in through, for "Sign in with
	    SSO"; `no_sso_connection` when its domain has none. */
	discoverSSO: (email: string) =>
		request<{ connection: SSOConnection; enforced: boolean }>('/account/sso/discover', {
			method: 'POST',
			body: { email }
		}),

	/** The languages this installation offers, and the one somebody gets
	    before they have chosen. The text itself is built into this app. */
	languages: (fetch?: Fetch) =>
		request<{ languages: PublicLanguage[]; default: string }>('/account/languages', { fetch }),

	/** One offered language's text, every key filled in: its own where it
	    has one, the base language's where it does not. */
	languageText: (code: string, fetch?: Fetch) =>
		request<{ language: PublicLanguage; messages: Record<string, string> }>(
			`/account/languages/${encodeURIComponent(code)}`,
			{ fetch }
		),

	/** What the flow behind this sign-in lets these pages offer. The handle
	    names the application whose flow applies; without one it is the
	    installation's default. */
	loginOptions: (handle: string, fetch?: Fetch) =>
		request<{ login: LoginOptions }>(
			handle ? `/account/login-options?request=${encode(handle)}` : '/account/login-options',
			{ fetch }
		),

	login: (body: { request: string; email: string; password: string; remember: boolean }) =>
		request<SignedIn>('/account/login', { method: 'POST', body }),

	register: (body: RegisterInput) =>
		request<SignedIn>('/account/register', { method: 'POST', body }),

	/** Finishes a sign-in that was waiting for an emailed code. It answers as
	    signing in does: where to go next, or nothing when there is nowhere. */
	code: (body: { handle: string; code: string }) =>
		request<SignedIn>('/account/login/code', { method: 'POST', body }),

	/** Sends another code for a sign-in that is still waiting. The handle
	    stays as it is, and so do the guesses already spent. */
	resendCode: (body: { handle: string }) =>
		request<{ code: CodeChallenge }>('/account/login/code/resend', { method: 'POST', body }),

	/** `language` is the one the page is shown in: the email is written in it. */
	forgotPassword: (body: { request: string; email: string; language: string }) =>
		request<{ status: string }>('/account/forgot-password', { method: 'POST', body }),

	checkReset: (token: string, fetch?: Fetch) =>
		request<{ valid: boolean }>(`/account/reset-password?token=${encode(token)}`, { fetch }),

	resetPassword: (body: { token: string; password: string }) =>
		request<{ status: string }>('/account/reset-password', { method: 'POST', body }),

	/** Uses a link sent to prove an address is the account's. */
	verifyEmail: (body: { token: string }) =>
		request<{ status: string }>('/account/verify-email', { method: 'POST', body }),

	logout: () => request<{ status: string }>('/account/logout', { method: 'POST' })
};

/** The signed-in user's own account. From a server load, pass its `fetch`:
    the session cookie goes along with it. */
export const account = {
	me: (fetch?: Fetch) => request<{ user: User }>('/account/me', { fetch }),

	updateProfile: (body: { first_name: string; last_name: string }) =>
		request<{ user: User }>('/account/me', { method: 'PATCH', body }),

	changePassword: (body: { current_password: string; new_password: string }) =>
		request<{ status: string }>('/account/password', { method: 'POST', body }),

	/** Starts moving the account to another sign-in address. The link goes to
	    the address typed, and nothing changes until it is opened — so this
	    answers the same whether or not somebody else already has it. */
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
