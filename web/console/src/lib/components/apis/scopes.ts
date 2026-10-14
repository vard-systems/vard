import type { API, APIInput, SigningAlgorithm } from '#lib/api/index.js';
import type { SelectOption } from '#lib/components/ui/index.js';

export type ScopeRow = {
	key: number;
	id?: string;
	name: string;
	description: string;
	is_default: boolean;
};

let nextKey = 0;

export function scopeRow(from?: Partial<ScopeRow>): ScopeRow {
	return { name: '', description: '', is_default: false, ...from, key: nextKey++ };
}

const tidyScope = (value: string) => value.trim().toLowerCase();

const pattern = /^[a-z][a-z0-9]*([:._-][a-z0-9]+)*$/;

const openid = ['openid', 'profile', 'email', 'offline_access', 'roles'];

export function scopeProblem(rows: ScopeRow[], row: ScopeRow): string | undefined {
	const value = tidyScope(row.name);

	if (value === '') return undefined;
	if (!pattern.test(value)) return 'Use lower case words joined by : . - or _, such as orders:read';
	if (openid.includes(value)) return 'That is an OpenID Connect scope';
	if (rows.filter((it) => tidyScope(it.name) === value).length > 1) return 'Listed twice';

	return undefined;
}

export function scopeInput(rows: ScopeRow[]): APIInput['scopes'] {
	return rows
		.filter((row) => tidyScope(row.name) !== '')
		.map((row) => ({
			id: row.id,
			name: tidyScope(row.name),
			description: row.description.trim(),
			is_default: row.is_default
		}));
}

export const algorithms: SelectOption<SigningAlgorithm>[] = [
	{
		value: 'RS256',
		label: 'RS256',
		description: 'RSA with SHA-256. Recommended: every JWT library supports it.'
	},
	{ value: 'PS256', label: 'PS256', description: 'RSA-PSS with SHA-256.' },
	{ value: 'ES256', label: 'ES256', description: 'ECDSA P-256 with SHA-256. Smaller tokens.' }
];

export function apiInput(api: API, changes: Partial<APIInput>): APIInput {
	return {
		name: api.name,
		identifier: api.identifier,
		description: api.description,
		enforce_roles: api.enforce_roles,
		signing_algorithm: api.signing_algorithm,
		token_lifetime: api.token_lifetime,
		allow_offline_access: api.allow_offline_access,
		scopes: api.scopes.map((scope) => ({ ...scope })),
		...changes
	};
}

export const roleBasedAccess = {
	label: 'Role-based access',
	description:
		"When enabled, users can receive only the scopes granted by their assigned roles. When disabled, access is determined by the application's allowed scopes."
};

export function lifetimeLabel(seconds: number): string {
	if (seconds <= 0) return "Application's";
	if (seconds % 3600 === 0) return `${seconds / 3600} h`;
	if (seconds % 60 === 0) return `${seconds / 60} min`;

	return `${seconds} s`;
}
