import type { ComponentType } from 'svelte';
import {
	RiAdminLine,
	RiAppsLine,
	RiCodeBoxLine,
	RiDatabase2Line,
	RiErrorWarningLine,
	RiFileList3Line,
	RiGitBranchLine,
	RiKey2Line,
	RiLockLine,
	RiLoginBoxLine,
	RiLogoutBoxRLine,
	RiMailLine,
	RiMailSendLine,
	RiPulseLine,
	RiShieldKeyholeLine,
	RiShieldUserLine,
	RiTranslate2,
	RiUserLine
} from 'svelte-remixicon';
import type { ActivityEvent } from '#lib/api/index.js';

/**
 * The activity log's actions ("<resource>.<event>", from audit.Record) as a sentence, icon and
 * tone, shared by every list of activity. Unlisted actions show their raw name.
 */

export type Category = 'access' | 'users' | 'roles' | 'applications' | 'apis' | 'admins' | 'other';

export type Tone = 'neutral' | 'success' | 'danger' | 'warning';

type Action = {
	label: string;
	verb: string;
	after?: string;
	category: Category;
	tone?: Tone;
	icon: ComponentType;
};

const actions: Record<string, Action> = {
	'admin.login': {
		label: 'Signed in',
		verb: 'signed in',
		category: 'access',
		tone: 'success',
		icon: RiLoginBoxLine
	},
	'admin.logout': {
		label: 'Signed out',
		verb: 'signed out',
		category: 'access',
		icon: RiLogoutBoxRLine
	},
	'admin.session_ended': {
		label: 'Session ended',
		verb: 'signed out one of their other sessions',
		category: 'access',
		icon: RiLogoutBoxRLine
	},
	'admin.sessions_ended': {
		label: 'Signed out elsewhere',
		verb: 'signed out of every other session',
		category: 'access',
		tone: 'warning',
		icon: RiLogoutBoxRLine
	},
	'admin.login_failed': {
		label: 'Sign-in failed',
		verb: 'failed to sign in',
		category: 'access',
		tone: 'danger',
		icon: RiErrorWarningLine
	},
	'admin.login_blocked': {
		label: 'Sign-in blocked',
		verb: 'was blocked from signing in',
		category: 'access',
		tone: 'warning',
		icon: RiLockLine
	},

	'admin.mfa_failed': {
		label: 'Two-factor failed',
		verb: 'entered a wrong two-factor code',
		category: 'access',
		tone: 'danger',
		icon: RiErrorWarningLine
	},
	'admin.mfa_enabled': {
		label: 'Two-factor on',
		verb: 'turned on two-factor sign-in',
		category: 'access',
		tone: 'success',
		icon: RiShieldKeyholeLine
	},
	'admin.mfa_replaced': {
		label: 'Authenticator replaced',
		verb: 'replaced their authenticator',
		category: 'access',
		tone: 'warning',
		icon: RiShieldKeyholeLine
	},
	'admin.mfa_disabled': {
		label: 'Two-factor off',
		verb: 'turned off two-factor sign-in',
		category: 'access',
		tone: 'warning',
		icon: RiShieldKeyholeLine
	},
	'admin.recovery_codes_regenerated': {
		label: 'Recovery codes renewed',
		verb: 'made new recovery codes',
		category: 'access',
		icon: RiKey2Line
	},
	'signing_keys.rotated': {
		label: 'Signing keys rotated',
		verb: 'rotated the token signing keys',
		category: 'admins',
		tone: 'warning',
		icon: RiKey2Line
	},
	'admin.mfa_reset': {
		label: 'Two-factor reset',
		verb: 'reset the two-factor sign-in of',
		category: 'admins',
		tone: 'warning',
		icon: RiShieldKeyholeLine
	},

	'user.login': {
		label: 'User signed in',
		verb: 'signed in to an application',
		category: 'access',
		tone: 'success',
		icon: RiLoginBoxLine
	},
	'user.logout': {
		label: 'User signed out',
		verb: 'signed out',
		category: 'access',
		icon: RiLogoutBoxRLine
	},
	'user.login_failed': {
		label: 'User sign-in failed',
		verb: 'failed to sign in to an application',
		category: 'access',
		tone: 'danger',
		icon: RiErrorWarningLine
	},
	'user.login_blocked': {
		label: 'User sign-in blocked',
		verb: 'was blocked from signing in',
		category: 'access',
		tone: 'warning',
		icon: RiLockLine
	},
	'user.registered': {
		label: 'User registered',
		verb: 'created an account',
		category: 'users',
		tone: 'success',
		icon: RiUserLine
	},
	'user.password_reset_requested': {
		label: 'Password reset requested',
		verb: 'asked to reset their password',
		category: 'access',
		icon: RiKey2Line
	},
	'user.password_changed_self': {
		label: 'Password changed',
		verb: 'changed their password',
		category: 'access',
		tone: 'warning',
		icon: RiKey2Line
	},
	'user.profile_updated': {
		label: 'Profile updated',
		verb: 'updated their profile',
		category: 'users',
		icon: RiUserLine
	},
	'user.session_revoked': {
		label: 'Device signed out',
		verb: 'signed out another device',
		category: 'access',
		icon: RiLogoutBoxRLine
	},
	'user.session_ended': {
		label: 'Session ended',
		verb: 'signed out a session of',
		category: 'access',
		icon: RiLogoutBoxRLine
	},
	'user.signed_out_everywhere': {
		label: 'Signed out everywhere',
		verb: 'signed out everywhere',
		category: 'access',
		tone: 'danger',
		icon: RiLogoutBoxRLine
	},
	'user.application_disconnected': {
		label: 'App disconnected',
		verb: 'disconnected an application',
		category: 'access',
		icon: RiAppsLine
	},
	'user.password_reset': {
		label: 'Password reset',
		verb: 'reset their password',
		category: 'access',
		tone: 'warning',
		icon: RiKey2Line
	},

	'user.created': {
		label: 'User created',
		verb: 'created user',
		category: 'users',
		tone: 'success',
		icon: RiUserLine
	},
	'user.updated': {
		label: 'User updated',
		verb: 'updated user',
		category: 'users',
		icon: RiUserLine
	},
	'user.deleted': {
		label: 'User deleted',
		verb: 'deleted user',
		category: 'users',
		tone: 'danger',
		icon: RiUserLine
	},
	'user.password_changed': {
		label: 'Password changed',
		verb: 'changed the password of',
		category: 'users',
		tone: 'warning',
		icon: RiKey2Line
	},
	'user.roles_assigned': {
		label: 'Roles assigned',
		verb: 'assigned roles to',
		category: 'roles',
		icon: RiShieldUserLine
	},
	'user.role_unassigned': {
		label: 'Role removed',
		verb: 'removed a role from',
		category: 'roles',
		icon: RiShieldUserLine
	},
	'user_field.created': {
		label: 'Field added',
		verb: 'added the user field',
		category: 'users',
		icon: RiFileList3Line
	},
	'user_field.updated': {
		label: 'Field updated',
		verb: 'updated the user field',
		category: 'users',
		icon: RiFileList3Line
	},
	'user_field.deleted': {
		label: 'Field removed',
		verb: 'removed the user field',
		category: 'users',
		tone: 'danger',
		icon: RiFileList3Line
	},

	'user_role.created': {
		label: 'Role created',
		verb: 'created role',
		category: 'roles',
		tone: 'success',
		icon: RiShieldUserLine
	},
	'user_role.updated': {
		label: 'Role updated',
		verb: 'updated role',
		category: 'roles',
		icon: RiShieldUserLine
	},
	'user_role.deleted': {
		label: 'Role deleted',
		verb: 'deleted role',
		category: 'roles',
		tone: 'danger',
		icon: RiShieldUserLine
	},

	'application.created': {
		label: 'Application registered',
		verb: 'registered application',
		category: 'applications',
		tone: 'success',
		icon: RiAppsLine
	},
	'application.updated': {
		label: 'Application updated',
		verb: 'updated application',
		category: 'applications',
		icon: RiAppsLine
	},
	'application.deleted': {
		label: 'Application deleted',
		verb: 'deleted application',
		category: 'applications',
		tone: 'danger',
		icon: RiAppsLine
	},
	'application.secret_rotated': {
		label: 'Secret rotated',
		verb: 'rotated the client secret of',
		category: 'applications',
		tone: 'warning',
		icon: RiKey2Line
	},
	'application.api_authorized': {
		label: 'API access granted',
		verb: 'gave',
		after: 'access to an API',
		category: 'apis',
		tone: 'success',
		icon: RiCodeBoxLine
	},
	'application.api_revoked': {
		label: 'API access revoked',
		verb: 'revoked the API access of',
		category: 'apis',
		tone: 'danger',
		icon: RiCodeBoxLine
	},

	'api.created': {
		label: 'API registered',
		verb: 'registered API',
		category: 'apis',
		tone: 'success',
		icon: RiCodeBoxLine
	},
	'api.updated': {
		label: 'API updated',
		verb: 'updated API',
		category: 'apis',
		icon: RiCodeBoxLine
	},
	'api.deleted': {
		label: 'API deleted',
		verb: 'deleted API',
		category: 'apis',
		tone: 'danger',
		icon: RiCodeBoxLine
	},

	'admin.created': {
		label: 'Administrator added',
		verb: 'added administrator',
		category: 'admins',
		tone: 'success',
		icon: RiAdminLine
	},
	'admin.updated': {
		label: 'Administrator updated',
		verb: 'updated administrator',
		category: 'admins',
		icon: RiAdminLine
	},
	'admin.profile_updated': {
		label: 'Profile updated',
		verb: 'updated their own profile',
		category: 'admins',
		icon: RiAdminLine
	},
	'admin.deleted': {
		label: 'Administrator removed',
		verb: 'removed administrator',
		category: 'admins',
		tone: 'danger',
		icon: RiAdminLine
	},
	'admin.password_changed': {
		label: 'Admin password changed',
		verb: 'changed the password of administrator',
		category: 'admins',
		tone: 'warning',
		icon: RiKey2Line
	},
	'admin_role.created': {
		label: 'Admin role created',
		verb: 'created admin role',
		category: 'admins',
		icon: RiShieldKeyholeLine
	},
	'admin_role.updated': {
		label: 'Admin role updated',
		verb: 'updated admin role',
		category: 'admins',
		icon: RiShieldKeyholeLine
	},
	'admin_role.deleted': {
		label: 'Admin role deleted',
		verb: 'deleted admin role',
		category: 'admins',
		tone: 'danger',
		icon: RiShieldKeyholeLine
	},
	'cache.entry_updated': {
		label: 'Cached value edited',
		verb: 'edited a cached value in the Redis database',
		category: 'other',
		tone: 'warning',
		icon: RiDatabase2Line
	},
	'cache.entry_deleted': {
		label: 'Cached key removed',
		verb: 'removed a cached key from the Redis database',
		category: 'other',
		icon: RiDatabase2Line
	},
	'cache.group_cleared': {
		label: 'Cache group cleared',
		verb: 'cleared a cache group in the Redis database',
		category: 'other',
		icon: RiDatabase2Line
	},
	'cache.flushed': {
		label: 'Redis database flushed',
		verb: 'flushed the Redis database',
		category: 'other',
		tone: 'danger',
		icon: RiDatabase2Line
	},
	'login_flow.created': {
		label: 'Login flow created',
		verb: 'created login flow',
		category: 'access',
		icon: RiGitBranchLine
	},
	'login_flow.updated': {
		label: 'Login flow updated',
		verb: 'updated login flow',
		category: 'access',
		icon: RiGitBranchLine
	},
	'login_flow.deleted': {
		label: 'Login flow deleted',
		verb: 'deleted login flow',
		category: 'access',
		tone: 'danger',
		icon: RiGitBranchLine
	},
	'sso_connection.created': {
		label: 'SSO connection added',
		verb: 'added SSO connection',
		category: 'access',
		icon: RiShieldKeyholeLine
	},
	'sso_connection.updated': {
		label: 'SSO connection updated',
		verb: 'updated SSO connection',
		category: 'access',
		icon: RiShieldKeyholeLine
	},
	'sso_connection.deleted': {
		label: 'SSO connection removed',
		verb: 'removed SSO connection',
		category: 'access',
		tone: 'danger',
		icon: RiShieldKeyholeLine
	},
	'sso_connection.sign_in_failed': {
		label: 'SSO sign-in failed',
		verb: 'could not complete a sign-in through SSO connection',
		category: 'access',
		tone: 'danger',
		icon: RiShieldKeyholeLine
	},
	'user.roles_synced': {
		label: 'Roles synced from SSO',
		verb: 'had roles synced from their identity provider',
		category: 'roles',
		icon: RiShieldUserLine
	},
	'language.created': {
		label: 'Language added',
		verb: 'added language',
		category: 'other',
		icon: RiTranslate2
	},
	'language.updated': {
		label: 'Language updated',
		verb: 'updated language',
		category: 'other',
		icon: RiTranslate2
	},
	'language.translated': {
		label: 'Translation saved',
		verb: 'saved the text of language',
		category: 'other',
		icon: RiTranslate2
	},
	'language.deleted': {
		label: 'Language removed',
		verb: 'removed language',
		category: 'other',
		tone: 'danger',
		icon: RiTranslate2
	},
	'mail.settings_updated': {
		label: 'Mail settings changed',
		verb: 'changed',
		category: 'other',
		icon: RiMailLine
	},
	'mail.content_updated': {
		label: 'Email content changed',
		verb: 'changed',
		category: 'other',
		icon: RiMailLine
	},
	'mail.test_sent': {
		label: 'Test email sent',
		verb: 'sent a test email through',
		category: 'other',
		tone: 'success',
		icon: RiMailSendLine
	},
	'mail.test_failed': {
		label: 'Test email failed',
		verb: 'could not send a test email through',
		category: 'other',
		tone: 'warning',
		icon: RiMailSendLine
	},
	'otp.settings_updated': {
		label: 'One-time codes changed',
		verb: 'changed',
		category: 'access',
		icon: RiKey2Line
	},
	'user.login_code_sent': {
		label: 'Code emailed',
		verb: 'was emailed a one-time code',
		category: 'access',
		icon: RiKey2Line
	},
	'user.login_code_failed': {
		label: 'Wrong code',
		verb: 'typed a one-time code that was not the one sent',
		category: 'access',
		tone: 'warning',
		icon: RiKey2Line
	}
};

const unnamed: Record<string, string> = {
	user: 'a user',
	user_field: 'a field',
	user_role: 'a role',
	application: 'an application',
	api: 'an API',
	admin_user: 'an administrator',
	admin_role: 'an admin role',
	login_flow: 'a login flow',
	language: 'a language',
	sso_connection: 'an SSO connection',
	mail: 'the mail settings',
	mail_content: 'the email content',
	otp: 'the one-time code settings'
};

export type Described = {
	label: string;
	actor: string;
	verb: string;
	subject: string;
	named: boolean;
	after: string;
	detail: string;
	category: Category;
	tone: Tone;
	icon: ComponentType;
};

const selfTargeted = new Set([
	'admin.login',
	'admin.logout',
	'admin.session_ended',
	'admin.sessions_ended',
	'admin.login_failed',
	'admin.login_blocked',
	'admin.mfa_failed',
	'admin.mfa_enabled',
	'admin.mfa_replaced',
	'admin.mfa_disabled',
	'admin.recovery_codes_regenerated',
	'user.login',
	'user.logout',
	'user.login_failed',
	'user.login_blocked',
	'user.registered',
	'user.password_reset_requested',
	'user.password_reset',
	'user.password_changed_self',
	'user.profile_updated',
	'user.session_revoked',
	'user.application_disconnected',
	'user.login_code_sent',
	'user.login_code_failed'
]);

export function describe(event: ActivityEvent): Described {
	const known = actions[event.action];
	const target = event.target;

	let subject = '';
	let named = false;
	if (target && !selfTargeted.has(event.action)) {
		named = Boolean(target.name);
		subject = target.name || (unnamed[target.type] ?? 'something');
	}

	if (!known) {
		return {
			label: event.action,
			actor: event.actor || 'System',
			verb: event.action,
			subject,
			named,
			after: '',
			detail: event.detail ?? '',
			category: 'other',
			tone: 'neutral',
			icon: RiPulseLine
		};
	}

	let after = known.after ?? '';
	let detail = event.detail ?? '';
	if (event.action === 'application.api_authorized' && detail) {
		after = `access to ${detail}`;
		detail = '';
	}

	return {
		label: known.label,
		actor: event.actor || 'System',
		verb: known.verb,
		subject,
		named,
		after,
		detail,
		category: known.category,
		tone: known.tone ?? 'neutral',
		icon: known.icon
	};
}

export const categoryLabels: Record<Category, string> = {
	access: 'Sign-ins',
	users: 'Users',
	roles: 'Roles',
	applications: 'Applications',
	apis: 'APIs',
	admins: 'Administration',
	other: 'Other'
};

export function actionsIn(category: Category): string[] {
	return Object.entries(actions)
		.filter(([, action]) => action.category === category)
		.map(([name]) => name);
}

export const refusedActions = [
	'admin.login_failed',
	'admin.login_blocked',
	'admin.mfa_failed',
	'user.login_failed',
	'user.login_blocked'
];
