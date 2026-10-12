import type { LogFilter } from '$lib/api';
import type { CacheKeyParams } from './cache';
import type { RoleListParams } from './roles';

/**
 * Query keys, built in one place so invalidation cannot miss a spelling. Everything under
 * `users.all` is invalidated when a user changes.
 */
export const keys = {
	/** The signed-in administrator's own account. */
	profile: {
		all: ['profile'] as const,
		sessions: ['profile', 'sessions'] as const,
		mfa: ['profile', 'mfa'] as const
	},

	users: {
		all: ['users'] as const,
		/** The list, as the search box and filter describe it, however many
		    pages of it are read. */
		list: (params: { search: string; verified: string; role: string }) =>
			['users', 'list', params.search, params.verified, params.role] as const,
		/** The fields a user record is made of. */
		fields: ['users', 'fields'] as const,
		/** Every role one user holds. */
		mappings: (id: string) => ['users', 'mappings', id] as const,
		/** A few users matching a typed name, for pickers. */
		search: (term: string) => ['users', 'search', term] as const
	},

	apis: {
		all: ['apis'] as const,
		list: (search: string) => ['apis', 'list', search] as const,
		one: (id: string) => ['apis', 'one', id] as const,
		/** The applications that may, or may not, use one API. */
		applications: (id: string) => ['apis', 'applications', id] as const,
		logs: (id: string) => ['apis', 'logs', id] as const
	},

	applications: {
		/** What one application may do with each API. */
		access: (id: string) => ['applications', 'access', id] as const,
		all: ['applications'] as const,
		/** The list, as the search box and filters describe it, however many
		    pages of it are read. */
		list: (params: { search: string; type: string }) =>
			['applications', 'list', params.search, params.type] as const,
		/** Every application the administrator can see, for pickers. */
		choices: ['applications', 'choices'] as const
	},

	roles: {
		all: ['roles'] as const,
		/** One tab of roles, as its filters describe it. */
		list: (params: RoleListParams) =>
			['roles', 'list', params.scope, params.application, params.search, params.isDefault] as const,
		/** Every role, for the pickers that offer them as choices. */
		choices: ['roles', 'choices'] as const
	},

	admins: {
		all: ['admins'] as const,
		/** The list, as the search box and filters describe it, however many
		    pages of it are read. */
		list: (params: { search: string; status: string; role: string }) =>
			['admins', 'list', params.search, params.status, params.role] as const,
		/** Every admin role. */
		roles: (search: string) => ['admins', 'roles', search] as const,
		/** The admin permission catalog, which never changes while running. */
		permissions: ['admins', 'permissions'] as const
	},

	organization: {
		/** There is one organisation, so this key is the whole of it. */
		settings: ['organization', 'settings'] as const
	},

	mail: {
		all: ['mail'] as const,
		/** There is one mail server, so this key is the whole of it. */
		settings: ['mail', 'settings'] as const,
		/** The words of every email, for every language. */
		content: ['mail', 'content'] as const
	},

	otp: {
		/** There is one record of how the emailed codes behave. */
		settings: ['otp', 'settings'] as const
	},

	cache: {
		/** Everything about Redis goes when any of it changes: a key removed
		    changes the counts, and a group cleared changes the keys. */
		all: ['cache'] as const,
		overview: ['cache', 'overview'] as const,
		/** One page of one database's keys, as the filters describe it. */
		keys: (params: CacheKeyParams) =>
			['cache', 'keys', params.database, params.kind, params.group, params.search] as const,
		key: (database: string, name: string) => ['cache', 'key', database, name] as const
	},

	social: {
		all: ['social'] as const,
		/** Every configured provider, and the kinds one may be. */
		providers: ['social', 'providers'] as const
	},

	flows: {
		all: ['flows'] as const,
		/** Every login flow, and the steps one can be made of. */
		list: ['flows', 'list'] as const
	},

	sso: {
		all: ['sso'] as const,
		/** Every connection. */
		list: ['sso', 'list'] as const
	},

	languages: {
		all: ['languages'] as const,
		/** Every language, with how much of each app it covers. */
		list: ['languages', 'list'] as const,
		/** Every language's text, and one language's text for one app. */
		translations: ['languages', 'translation'] as const,
		translation: (code: string, app: string) => ['languages', 'translation', code, app] as const
	},

	logs: {
		all: ['logs'] as const,
		/** One filtered view of the log, however many pages of it are read. */
		list: (filter: LogFilter) =>
			[
				'logs',
				'list',
				filter.q ?? '',
				(filter.actions ?? []).join(','),
				filter.actor ?? '',
				filter.from ?? '',
				filter.to ?? ''
			] as const
	},

	sessions: {
		all: ['sessions'] as const,
		/** The list, as the search box and the user filter describe it. */
		list: (params: { search: string; user: string }) =>
			['sessions', 'list', params.search, params.user] as const
	},

	admin: {
		/** The dashboard's counts and recent activity, which a change to
		    anything it counts invalidates. */
		overview: ['admin', 'overview'] as const
	}
} as const;
