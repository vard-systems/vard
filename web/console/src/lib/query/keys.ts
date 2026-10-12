import type { LogFilter } from '$lib/api';
import type { CacheKeyParams } from './cache';
import type { RoleListParams } from './roles';

/**
 * Query keys, built in one place so invalidation cannot miss a spelling. Everything under
 * `users.all` is invalidated when a user changes.
 */
export const keys = {
	profile: {
		all: ['profile'] as const,
		sessions: ['profile', 'sessions'] as const,
		mfa: ['profile', 'mfa'] as const
	},

	users: {
		all: ['users'] as const,
		list: (params: { search: string; verified: string; role: string }) =>
			['users', 'list', params.search, params.verified, params.role] as const,
		fields: ['users', 'fields'] as const,
		mappings: (id: string) => ['users', 'mappings', id] as const,
		search: (term: string) => ['users', 'search', term] as const
	},

	apis: {
		all: ['apis'] as const,
		list: (search: string) => ['apis', 'list', search] as const,
		one: (id: string) => ['apis', 'one', id] as const,
		applications: (id: string) => ['apis', 'applications', id] as const,
		logs: (id: string) => ['apis', 'logs', id] as const
	},

	applications: {
		access: (id: string) => ['applications', 'access', id] as const,
		all: ['applications'] as const,
		list: (params: { search: string; type: string }) =>
			['applications', 'list', params.search, params.type] as const,
		choices: ['applications', 'choices'] as const
	},

	roles: {
		all: ['roles'] as const,
		list: (params: RoleListParams) =>
			['roles', 'list', params.scope, params.application, params.search, params.isDefault] as const,
		choices: ['roles', 'choices'] as const
	},

	admins: {
		all: ['admins'] as const,
		list: (params: { search: string; status: string; role: string }) =>
			['admins', 'list', params.search, params.status, params.role] as const,
		roles: (search: string) => ['admins', 'roles', search] as const,
		permissions: ['admins', 'permissions'] as const
	},

	organization: {
		settings: ['organization', 'settings'] as const
	},

	mail: {
		all: ['mail'] as const,
		settings: ['mail', 'settings'] as const,
		content: ['mail', 'content'] as const
	},

	otp: {
		settings: ['otp', 'settings'] as const
	},

	cache: {
		all: ['cache'] as const,
		overview: ['cache', 'overview'] as const,
		keys: (params: CacheKeyParams) =>
			['cache', 'keys', params.database, params.kind, params.group, params.search] as const,
		key: (database: string, name: string) => ['cache', 'key', database, name] as const
	},

	social: {
		all: ['social'] as const,
		providers: ['social', 'providers'] as const
	},

	flows: {
		all: ['flows'] as const,
		list: ['flows', 'list'] as const
	},

	sso: {
		all: ['sso'] as const,
		list: ['sso', 'list'] as const
	},

	languages: {
		all: ['languages'] as const,
		list: ['languages', 'list'] as const,
		translations: ['languages', 'translation'] as const,
		translation: (code: string, app: string) => ['languages', 'translation', code, app] as const
	},

	logs: {
		all: ['logs'] as const,
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
		list: (params: { search: string; user: string }) =>
			['sessions', 'list', params.search, params.user] as const
	},

	admin: {
		overview: ['admin', 'overview'] as const
	}
} as const;
